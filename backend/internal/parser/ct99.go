package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GELogRecord representa un registro individual extraído del archivo de logs gesys_ct99.log.
type GELogRecord struct {
	SRID              string `json:"sr_id"`                   // Identificador del bloque: "SR 409"
	Date              string `json:"date"`                    // Fecha original: "Thu Sep 3 16:19:13 2026"
	Timestamp         string `json:"timestamp"`               // Fecha estándar: "2026-09-03 16:19:13"
	TimestampISO      string `json:"timestamp_iso"`           // Formato ISO-8601 UTC: "2026-09-03T16:19:13Z"
	Epoch             int64  `json:"epoch"`                   // Timestamp en formato Unix Epoch
	TimestampRaw      string `json:"timestamp_raw,omitempty"` // Texto original sin alteración: "Mon Sep 14 17:11:27 2026"
	ErmesCode         string `json:"ermes_code"`
	SeverityCode      int    `json:"severity_code"`
	Severity          string `json:"severity"` // "Info", "Pri/Soft" (Warn), "Pri/Hard" (Error), "Fatal"
	Host              string `json:"host"`
	Process           string `json:"process,omitempty"`
	SourceFile        string `json:"source_file,omitempty"`
	Device            string `json:"device,omitempty"`
	SerialNumber      string `json:"serial_number,omitempty"`
	TubeHealthIndex   string `json:"tube_health_index,omitempty"`
	TubeHeatLoad      string `json:"tube_heat_load,omitempty"`
	AccumulatedScans  string `json:"accumulated_scans,omitempty"`
	TotalSliceCount   string `json:"total_slice_count,omitempty"`
	MasAccumulated    string `json:"mas_accumulated,omitempty"`
	GantryRevolutions string `json:"gantry_revolutions,omitempty"`
	TubeTemperature   string `json:"tube_temperature,omitempty"`
	Message           string `json:"message"`
}

// Expresiones regulares precompiladas para optimizar el rendimiento sin recompilación
var (
	reBlockHeader = regexp.MustCompile(`^SR\s+(\d+)`)
	reBlockFooter = regexp.MustCompile(`^EN\s+(\d+)`)

	// Regex de extracción clave-valor en cuerpo
	reErmesHash         = regexp.MustCompile(`(?i)Ermes\s*#\s*:\s*(\w+)`)
	reDevice            = regexp.MustCompile(`(?i)Device\s*:\s*(.+)`)
	reSerialNumber      = regexp.MustCompile(`(?i)Serial\s+Number\s*:\s*(.+)`)
	reTubeHealthIndex   = regexp.MustCompile(`(?i)Tube\s+Health\s+Index\s*:\s*([\d\.]+\s*%?)`)
	reTubeHeatLoad      = regexp.MustCompile(`(?i)Tube\s+Heat\s+Load\s*:\s*([\d\.]+\s*%?)`)
	reScans             = regexp.MustCompile(`(?i)Accumulated\s+Scans\s*:\s*(\d+)`)
	reSlices            = regexp.MustCompile(`(?i)Total\s+Slice\s+Count\s*:\s*(\d+)`)
	reMasAccumulated    = regexp.MustCompile(`(?i)(?:reports\s+a\s+total\s+of\s+([\d\.]+)\s*ma\*?s|mAs\s*(?:Accumulated)?\s*:\s*([\d\.]+))`)
	reGantryRevolutions = regexp.MustCompile(`(?i)Total\s+No\.\s+of\s+Gantry\s+Revolutions\s*:\s*(\d+)`)
	reTubeTemp          = regexp.MustCompile(`(?i)Tube\s+temperature\s+.*?([\d\.]+)\s*degrees\s+Celsius`)

	// Regex para detectar fechas estándar (ej: "Mon Sep 14 17:11:27 2026" o "Thu Sep  3 16:25:00 2026")
	reDate = regexp.MustCompile(`(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun)\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d+\s+\d{2}:\d{2}:\d{2}\s+\d{4}`)
)

// MapSeverity traduce el código numérico a su etiqueta textual en estándar GE Healthcare.
func MapSeverity(code int) string {
	switch code {
	case 1:
		return "Info"
	case 2:
		return "Pri/Soft"
	case 3:
		return "Pri/Hard"
	case 4:
		return "Fatal"
	default:
		return "Unknown"
	}
}

// ParseAndFormatTimestamp analiza y formatea fechas tipo "Thu Sep  3 16:25:00 2026" o "Mon Sep 14 17:11:27 2026"
func ParseAndFormatTimestamp(raw string, existingEpoch int64) (string, string, string, int64) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if existingEpoch > 0 {
			t := time.Unix(existingEpoch, 0).UTC()
			return t.Format("2006-01-02 15:04:05"), t.Format(time.RFC3339), t.Format(time.ANSIC), existingEpoch
		}
		return "", "", "", 0
	}

	formats := []string{
		time.ANSIC,
		"Mon Jan _2 15:04:05 2006",
		"Mon Jan 02 15:04:05 2006",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	var parsedTime time.Time
	var err error
	for _, f := range formats {
		parsedTime, err = time.Parse(f, raw)
		if err == nil {
			break
		}
	}

	if err == nil {
		formatted := parsedTime.Format("2006-01-02 15:04:05")
		iso := parsedTime.Format(time.RFC3339)
		epoch := existingEpoch
		if epoch <= 0 {
			epoch = parsedTime.Unix()
		}
		return formatted, iso, raw, epoch
	}

	if existingEpoch > 0 {
		t := time.Unix(existingEpoch, 0).UTC()
		formatted := t.Format("2006-01-02 15:04:05")
		iso := t.Format(time.RFC3339)
		return formatted, iso, raw, existingEpoch
	}

	return raw, "", raw, existingEpoch
}

// ParseCT99Logs lee un reader línea por línea mediante bufio.Scanner
// procesando los bloques delimitados por SR [ID] y EN [ID].
func ParseCT99Logs(r io.Reader, maxRecords int) ([]GELogRecord, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var records []GELogRecord
	var currentID string
	var inBlock bool
	var blockLines []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inBlock {
			if matches := reBlockHeader.FindStringSubmatch(trimmed); len(matches) > 1 {
				inBlock = true
				currentID = matches[1]
				blockLines = []string{line}
			}
			continue
		}

		blockLines = append(blockLines, line)

		if matches := reBlockFooter.FindStringSubmatch(trimmed); len(matches) > 1 {
			footerID := matches[1]
			if footerID == currentID {
				record := parseSingleBlock(currentID, blockLines)
				records = append(records, record)

				if maxRecords > 0 && len(records) >= maxRecords {
					break
				}
			}
			inBlock = false
			currentID = ""
			blockLines = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("error escaneando archivo de log: %w", err)
	}

	return records, nil
}

// ParseCT99String procesa un texto string directamente.
func ParseCT99String(content string, maxRecords int) ([]GELogRecord, error) {
	return ParseCT99Logs(strings.NewReader(content), maxRecords)
}

// parseSingleBlock analiza el contenido de un bloque individual delimitado por SR/EN.
func parseSingleBlock(srID string, lines []string) GELogRecord {
	formattedSRID := srID
	if !strings.HasPrefix(srID, "SR ") {
		formattedSRID = "SR " + srID
	}

	rec := GELogRecord{
		SRID: formattedSRID,
	}

	var contentLines []string
	var nonMetaLines []string
	var rawDateFound string

	innerLines := lines
	if len(lines) > 2 {
		innerLines = lines[1 : len(lines)-1]
	}

	for i := 0; i < len(innerLines); i++ {
		rawLine := innerLines[i]
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" {
			continue
		}
		contentLines = append(contentLines, trimmed)
	}

	if len(contentLines) >= 3 {
		fields0 := strings.Split(contentLines[0], "\t")
		if len(fields0) < 5 {
			fields0 = strings.Fields(contentLines[0])
		}

		if len(fields0) >= 1 {
			if ep, err := strconv.ParseInt(fields0[0], 10, 64); err == nil {
				rec.Epoch = ep
			}
		}

		for j := len(fields0) - 1; j >= 0; j-- {
			val := strings.TrimSpace(fields0[j])
			if rec.SeverityCode == 0 {
				if sCode, err := strconv.Atoi(val); err == nil && sCode >= 1 && sCode <= 4 {
					rec.SeverityCode = sCode
					rec.Severity = MapSeverity(sCode)
					continue
				}
			}
			if rec.ErmesCode == "" {
				if len(val) >= 4 {
					rec.ErmesCode = val
				}
			}
		}

		if d := reDate.FindString(contentLines[0]); d != "" {
			rawDateFound = d
		}

		fields1 := strings.Split(contentLines[1], "\t")
		if len(fields1) >= 2 {
			rec.Process = strings.TrimSpace(fields1[len(fields1)-1])
			rec.Host = strings.TrimSpace(fields1[0])
		} else {
			fields1 = strings.Fields(contentLines[1])
			if len(fields1) >= 2 {
				rec.Process = strings.TrimSpace(fields1[len(fields1)-1])
				rec.Host = strings.TrimSpace(fields1[0])
			} else if len(fields1) == 1 {
				rec.Process = fields1[0]
				rec.Host = fields1[0]
			}
		}

		rec.SourceFile = strings.Join(strings.Fields(contentLines[2]), ":")

		innerLines = innerLines[3:]
	}

	for _, rawLine := range innerLines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		if d := reDate.FindString(line); d != "" {
			if rawDateFound == "" || strings.TrimSpace(line) == strings.TrimSpace(d) {
				rawDateFound = d
			}
			if strings.TrimSpace(line) == strings.TrimSpace(d) {
				continue
			}
		}

		if strings.Contains(line, "Ermes # :") || strings.Contains(line, "Host :") {
			if rec.ErmesCode == "" {
				if m := reErmesHash.FindStringSubmatch(line); len(m) > 1 {
					rec.ErmesCode = strings.TrimSpace(m[1])
				}
			}
			if strings.Contains(line, "Host :") {
				parts := strings.Split(line, "Host :")
				if len(parts) > 1 {
					sub := strings.Split(parts[1], "Ermes")[0]
					if s := strings.TrimSpace(sub); s != "" {
						rec.Host = s
					}
				}
			}
			continue
		}

		if rec.ErmesCode == "" {
			if m := reErmesHash.FindStringSubmatch(line); len(m) > 1 {
				rec.ErmesCode = strings.TrimSpace(m[1])
				continue
			}
		}

		if m := reDevice.FindStringSubmatch(line); len(m) > 1 {
			rec.Device = strings.TrimSpace(m[1])
			continue
		}

		if m := reSerialNumber.FindStringSubmatch(line); len(m) > 1 {
			rec.SerialNumber = strings.TrimSpace(m[1])
			continue
		}

		if m := reTubeHealthIndex.FindStringSubmatch(line); len(m) > 1 {
			rec.TubeHealthIndex = strings.TrimSpace(m[1])
			continue
		}

		if m := reTubeHeatLoad.FindStringSubmatch(line); len(m) > 1 {
			rec.TubeHeatLoad = strings.TrimSpace(m[1])
			continue
		}

		if m := reScans.FindStringSubmatch(line); len(m) > 1 {
			rec.AccumulatedScans = strings.TrimSpace(m[1])
			continue
		}

		if m := reSlices.FindStringSubmatch(line); len(m) > 1 {
			rec.TotalSliceCount = strings.TrimSpace(m[1])
			continue
		}

		if m := reMasAccumulated.FindStringSubmatch(line); len(m) > 1 {
			val := m[1]
			if val == "" && len(m) > 2 {
				val = m[2]
			}
			rec.MasAccumulated = strings.TrimSpace(val)
			continue
		}

		if m := reGantryRevolutions.FindStringSubmatch(line); len(m) > 1 {
			rec.GantryRevolutions = strings.TrimSpace(m[1])
			continue
		}

		if m := reTubeTemp.FindStringSubmatch(line); len(m) > 1 {
			rec.TubeTemperature = strings.TrimSpace(m[1])
			// Preservar la línea completa en el mensaje si describe calentamiento
			if strings.Contains(line, "Warmup") || strings.Contains(line, "Warm up") || strings.Contains(line, "WarmupII") {
				nonMetaLines = append(nonMetaLines, line)
				continue
			}
			continue
		}

		nonMetaLines = append(nonMetaLines, line)
	}

	formatted, iso, raw, epoch := ParseAndFormatTimestamp(rawDateFound, rec.Epoch)
	rec.Date = raw
	rec.Timestamp = formatted
	rec.TimestampISO = iso
	rec.TimestampRaw = raw
	rec.Epoch = epoch

	rec.Message = strings.Join(nonMetaLines, " ")

	return rec
}
