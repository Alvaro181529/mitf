package parser

import (
	"math"
	"net/http"
	"strconv"
	"strings"
)

// FilterOptions mapea las opciones de búsqueda y paginación recibidas desde la petición HTTP.
type FilterOptions struct {
	Severity string `json:"severity"` // Filtra por nivel numérico (ej: "2", "3") o texto ("Pri/Soft", "Error", "Warn")
	Host     string `json:"host"`     // Filtra por el host o subsistema (ej: "tubemgr", "tgp", "Table Gantry Processor")
	Search   string `json:"search"`   // Búsqueda por coincidencia de texto en mensaje, código Ermes, archivo .cxx, SR_ID o dispositivo
	Device   string `json:"device"`   // Filtra por componente de hardware (ej: "X-Ray Tube")
	Date     string `json:"date"`     // Filtra por fecha (ej: "2026-09-14", "Sep  3", etc.)
	Page     int    `json:"page"`     // Número de página actual (Default: 1)
	Limit    int    `json:"limit"`    // Cantidad de registros por página (Default: 20, Max: 100)
}

// PaginationMeta contiene los metadatos de paginación para el frontend (Next.js / React).
type PaginationMeta struct {
	TotalRecords int  `json:"total_records"`
	CurrentPage  int  `json:"current_page"`
	Limit        int  `json:"limit"`
	TotalPages   int  `json:"total_pages"`
	HasNext      bool `json:"has_next"`
	HasPrev      bool `json:"has_prev"`
}

// PaginatedLogsResponse es la estructura estándar devuelta a la API.
type PaginatedLogsResponse struct {
	Data       []GELogRecord  `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// ParseFilterOptions extrae y valida los parámetros query desde la petición HTTP (*http.Request).
func ParseFilterOptions(r *http.Request) FilterOptions {
	q := r.URL.Query()

	page := 1
	if pStr := q.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			if l > 100 {
				limit = 100
			} else {
				limit = l
			}
		}
	}

	return FilterOptions{
		Severity: strings.TrimSpace(q.Get("severity")),
		Host:     strings.TrimSpace(q.Get("host")),
		Device:   strings.TrimSpace(q.Get("device")),
		Search:   strings.TrimSpace(q.Get("search")),
		Date:     strings.TrimSpace(q.Get("date")),
		Page:     page,
		Limit:    limit,
	}
}

// ApplyFilters evalúa las condiciones de búsqueda en memoria mediante comprobaciones case-insensitive.
func ApplyFilters(records []GELogRecord, opts FilterOptions) []GELogRecord {
	if opts.Severity == "" && opts.Host == "" && opts.Device == "" && opts.Search == "" && opts.Date == "" {
		return records
	}

	severityLower := strings.ToLower(opts.Severity)
	hostLower := strings.ToLower(opts.Host)
	deviceLower := strings.ToLower(opts.Device)
	searchLower := strings.ToLower(opts.Search)
	dateLower := strings.ToLower(opts.Date)

	var filtered []GELogRecord

	for _, rec := range records {
		// 1. Filtro por Severity (soporta código numérico o nombres comunes GE)
		if severityLower != "" {
			matchSeverity := false
			if strconv.Itoa(rec.SeverityCode) == severityLower {
				matchSeverity = true
			} else if strings.Contains(strings.ToLower(rec.Severity), severityLower) {
				matchSeverity = true
			} else {
				switch severityLower {
				case "1", "info", "diag":
					matchSeverity = rec.SeverityCode == 1
				case "2", "warn", "warning", "pri/soft", "soft":
					matchSeverity = rec.SeverityCode == 2 ||
						strings.Contains(strings.ToLower(rec.Severity), "soft") ||
						strings.Contains(strings.ToLower(rec.Severity), "warn")
				case "3", "error", "pri/hard", "hard":
					matchSeverity = rec.SeverityCode == 3 ||
						strings.Contains(strings.ToLower(rec.Severity), "hard") ||
						strings.Contains(strings.ToLower(rec.Severity), "error")
				case "4", "fatal":
					matchSeverity = rec.SeverityCode == 4 ||
						strings.Contains(strings.ToLower(rec.Severity), "fatal")
				}
			}
			if !matchSeverity {
				continue
			}
		}

		// 2. Filtro por Host / Subsistema
		if hostLower != "" {
			if !strings.Contains(strings.ToLower(rec.Host), hostLower) &&
				!strings.Contains(strings.ToLower(rec.Process), hostLower) {
				continue
			}
		}

		// 3. Filtro por Device
		if deviceLower != "" {
			if !strings.Contains(strings.ToLower(rec.Device), deviceLower) {
				continue
			}
		}

		// 4. Filtro por Fecha (comprueba en fecha formateada, ISO, raw o campo date)
		if dateLower != "" {
			if !strings.Contains(strings.ToLower(rec.Timestamp), dateLower) &&
				!strings.Contains(strings.ToLower(rec.TimestampRaw), dateLower) &&
				!strings.Contains(strings.ToLower(rec.Date), dateLower) {
				continue
			}
		}

		// 5. Búsqueda por coincidencia de texto (Search)
		// Mensaje, código Ermes, archivo .cxx, SR_ID o dispositivo
		if searchLower != "" {
			matchSearch := strings.Contains(strings.ToLower(rec.Message), searchLower) ||
				strings.Contains(strings.ToLower(rec.ErmesCode), searchLower) ||
				strings.Contains(strings.ToLower(rec.SourceFile), searchLower) ||
				strings.Contains(strings.ToLower(rec.SRID), searchLower) ||
				strings.Contains(strings.ToLower(rec.Device), searchLower)
			if !matchSearch {
				continue
			}
		}

		filtered = append(filtered, rec)
	}

	return filtered
}

// PaginateRecords corta el slice filtrado usando la fórmula start = (page - 1) * limit
// evitando errores de index out of range.
func PaginateRecords(records []GELogRecord, page, limit int) ([]GELogRecord, PaginationMeta) {
	total := len(records)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	startIndex := (page - 1) * limit
	if startIndex >= total {
		return []GELogRecord{}, PaginationMeta{
			TotalRecords: total,
			CurrentPage:  page,
			Limit:        limit,
			TotalPages:   totalPages,
			HasNext:      false,
			HasPrev:      page > 1,
		}
	}

	endIndex := startIndex + limit
	if endIndex > total {
		endIndex = total
	}

	paged := records[startIndex:endIndex]

	return paged, PaginationMeta{
		TotalRecords: total,
		CurrentPage:  page,
		Limit:        limit,
		TotalPages:   totalPages,
		HasNext:      page < totalPages,
		HasPrev:      page > 1,
	}
}
