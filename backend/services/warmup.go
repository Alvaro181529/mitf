package services

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mitfv2/internal/parser"
)

// Tipos de Rutinas de Calentamiento del Tubo de Rayos X
const (
	RoutineTypeColdWarmup      = "COLD_TUBE_WARMUP"
	RoutineTypeWarmupII        = "WARMUP_II"
	RoutineTypeFastCalibration = "FAST_CALIBRATION"
	RoutineTypeSkipped         = "SKIPPED_WARMUP"
)

// Regex precompiladas para extracción de métricas de calentamiento
var (
	reBeforeCold = regexp.MustCompile(`(?i)Tube\s+temperature\s+before\s+Cold\s+Tube\s+Warmup(?:\s*\((Fast\s+Calibration)\))?\s+is\s+([\d\.]+)\s*degrees\s+Celsius`)
	reAfterCold  = regexp.MustCompile(`(?i)Tube\s+temperature\s+after\s+Cold\s+Tube\s+Warmup\s+is\s+([\d\.]+)\s*degrees\s+Celsius`)
	reWarmupII   = regexp.MustCompile(`(?i)Tube\s+temperature\s+before\s+WarmupII\s+is\s+([\d\.]+)\s*degrees\s+Celsius`)
	reSkipped    = regexp.MustCompile(`(?i)Tube\s+warm\s*up\s+was\s+skipped.*?exam\s*(\d+).*?Max\s+mA\s+for\s+120kV\s*=\s*(\d+).*?Max\s+mA\s+for\s+140kV\s*=\s*(\d+)`)
)

// WarmupRoutineCycle representa una rutina de calentamiento completa del tubo RX.
type WarmupRoutineCycle struct {
	ID                 int     `json:"id"`
	RoutineType        string  `json:"routine_type"`
	Status             string  `json:"status"` // "COMPLETED", "INCOMPLETE"
	StartTime          string  `json:"start_time"`
	StartTimeISO       string  `json:"start_time_iso"`
	StartEpoch         int64   `json:"start_epoch"`
	EndTime            string  `json:"end_time,omitempty"`
	EndTimeISO         string  `json:"end_time_iso,omitempty"`
	EndEpoch           int64   `json:"end_epoch,omitempty"`
	DurationSeconds    int     `json:"duration_seconds"`
	InitialTempCelsius float64 `json:"initial_temp_celsius"`
	FinalTempCelsius   float64 `json:"final_temp_celsius"`
	TempRiseCelsius    float64 `json:"temp_rise_celsius"`
	Process            string  `json:"process"`
	StartErmesCode     string  `json:"start_ermes_code"`
	EndErmesCode       string  `json:"end_ermes_code,omitempty"`
	StartSRID          string  `json:"start_sr_id,omitempty"`
	EndSRID            string  `json:"end_sr_id,omitempty"`
	Details            string  `json:"details"`
}

// SkippedWarmupEvent registra una omisión de calentamiento que forzó limitación de corriente mA.
type SkippedWarmupEvent struct {
	SRID         string `json:"sr_id"`
	Date         string `json:"date"`
	TimestampISO string `json:"timestamp_iso"`
	Epoch        int64  `json:"epoch"`
	ExamID       string `json:"exam_id"`
	MaxMa120kV   int    `json:"max_ma_120kv"`
	MaxMa140kV   int    `json:"max_ma_140kv"`
	Severity     string `json:"severity"`
	ErmesCode    string `json:"ermes_code"`
	Process      string `json:"process"`
	Message      string `json:"message"`
}

// WarmupSummary agrupa los KPIs globales de rutinas de calentamiento.
type WarmupSummary struct {
	TotalCompletedRoutines int                 `json:"total_completed_routines"`
	TotalSkippedEvents     int                 `json:"total_skipped_events"`
	LastRoutineDate        string              `json:"last_routine_date,omitempty"`
	LastRoutineDateISO     string              `json:"last_routine_date_iso,omitempty"`
	LastInitialTempCelsius float64             `json:"last_initial_temp_celsius"`
	LastFinalTempCelsius   float64             `json:"last_final_temp_celsius"`
	LastTempRiseCelsius    float64             `json:"last_temp_rise_celsius"`
	AverageTempRiseCelsius float64             `json:"average_temp_rise_celsius"`
	AverageDurationSeconds float64             `json:"average_duration_seconds"`
	TubeThermalStatus      string              `json:"tube_thermal_status"` // "READY", "NEEDS_WARMUP", "OVERHEATED"
	ComplianceStatus       string              `json:"compliance_status"`   // "COMPLIANT", "ATTENTION_REQUIRED"
	LatestSkippedEvent     *SkippedWarmupEvent `json:"latest_skipped_event,omitempty"`
}

// TubeWarmupResponse estructura devuelta por GET /api/v1/hardware/tube-warmup.
type TubeWarmupResponse struct {
	Status        string                 `json:"status"`
	Timestamp     string                 `json:"timestamp"`
	ServerInfo    map[string]interface{} `json:"server_info,omitempty"`
	Summary       WarmupSummary          `json:"summary"`
	Routines      []WarmupRoutineCycle   `json:"routines"`
	SkippedEvents []SkippedWarmupEvent   `json:"skipped_events"`
}

// ParseWarmupRecords procesa un conjunto de registros GELogRecord y construye las rutinas y eventos de calentamiento.
func ParseWarmupRecords(records []parser.GELogRecord) ([]WarmupRoutineCycle, []SkippedWarmupEvent, WarmupSummary) {
	// 1. Filtrar únicamente eventos relevantes para calentamiento
	type rawPoint struct {
		isBefore  bool
		isAfter   bool
		isSkip    bool
		isWarmII  bool
		isFastCal bool
		temp      float64
		record    parser.GELogRecord
		examID    string
		ma120     int
		ma140     int
	}

	var points []rawPoint
	var skippedList []SkippedWarmupEvent

	for _, rec := range records {
		msg := rec.Message

		// Caso A: Before Cold Tube Warmup
		if m := reBeforeCold.FindStringSubmatch(msg); len(m) > 2 {
			tempVal, _ := strconv.ParseFloat(m[2], 64)
			isFast := strings.Contains(strings.ToLower(m[1]), "fast")
			points = append(points, rawPoint{
				isBefore:  true,
				isFastCal: isFast,
				temp:      tempVal,
				record:    rec,
			})
			continue
		}

		// Caso B: After Cold Tube Warmup
		if m := reAfterCold.FindStringSubmatch(msg); len(m) > 1 {
			tempVal, _ := strconv.ParseFloat(m[1], 64)
			points = append(points, rawPoint{
				isAfter: true,
				temp:    tempVal,
				record:  rec,
			})
			continue
		}

		// Caso C: Warmup II
		if m := reWarmupII.FindStringSubmatch(msg); len(m) > 1 {
			tempVal, _ := strconv.ParseFloat(m[1], 64)
			points = append(points, rawPoint{
				isWarmII: true,
				temp:     tempVal,
				record:   rec,
			})
			continue
		}

		// Caso D: Skipped Warm-up
		if m := reSkipped.FindStringSubmatch(msg); len(m) > 3 {
			ma120Val, _ := strconv.Atoi(m[2])
			ma140Val, _ := strconv.Atoi(m[3])
			skippedList = append(skippedList, SkippedWarmupEvent{
				SRID:         rec.SRID,
				Date:         rec.Date,
				TimestampISO: rec.TimestampISO,
				Epoch:        rec.Epoch,
				ExamID:       m[1],
				MaxMa120kV:   ma120Val,
				MaxMa140kV:   ma140Val,
				Severity:     rec.Severity,
				ErmesCode:    rec.ErmesCode,
				Process:      rec.Process,
				Message:      strings.TrimSpace(rec.Message),
			})
		}
	}

	// Ordenar cronológicamente por Epoch/Timestamp
	sort.Slice(points, func(i, j int) bool {
		if points[i].record.Epoch != points[j].record.Epoch {
			return points[i].record.Epoch < points[j].record.Epoch
		}
		return points[i].record.Date < points[j].record.Date
	})

	var routines []WarmupRoutineCycle
	var pendingBefore *rawPoint
	routineID := 1

	for _, pt := range points {
		if pt.isBefore {
			// Si ya había un before pendiente sin after, registrar como incompleto o reemplazar
			if pendingBefore != nil {
				routines = append(routines, WarmupRoutineCycle{
					ID:                 routineID,
					RoutineType:        RoutineTypeColdWarmup,
					Status:             "INCOMPLETE",
					StartTime:          pendingBefore.record.Date,
					StartTimeISO:       pendingBefore.record.TimestampISO,
					StartEpoch:         pendingBefore.record.Epoch,
					InitialTempCelsius: pendingBefore.temp,
					FinalTempCelsius:   pendingBefore.temp,
					TempRiseCelsius:    0,
					Process:            pendingBefore.record.Process,
					StartErmesCode:     pendingBefore.record.ErmesCode,
					StartSRID:          pendingBefore.record.SRID,
					Details:            fmt.Sprintf("Calentamiento iniciado a %.1f°C sin registro de finalización", pendingBefore.temp),
				})
				routineID++
			}
			curr := pt
			pendingBefore = &curr
			continue
		}

		if pt.isAfter && pendingBefore != nil {
			dur := int(pt.record.Epoch - pendingBefore.record.Epoch)
			if dur < 0 {
				dur = 0
			}
			rise := math.Round((pt.temp-pendingBefore.temp)*100) / 100

			routType := RoutineTypeColdWarmup
			if pendingBefore.isFastCal {
				routType = RoutineTypeFastCalibration
			}

			routines = append(routines, WarmupRoutineCycle{
				ID:                 routineID,
				RoutineType:        routType,
				Status:             "COMPLETED",
				StartTime:          pendingBefore.record.Date,
				StartTimeISO:       pendingBefore.record.TimestampISO,
				StartEpoch:         pendingBefore.record.Epoch,
				EndTime:            pt.record.Date,
				EndTimeISO:         pt.record.TimestampISO,
				EndEpoch:           pt.record.Epoch,
				DurationSeconds:    dur,
				InitialTempCelsius: pendingBefore.temp,
				FinalTempCelsius:   pt.temp,
				TempRiseCelsius:    rise,
				Process:            pt.record.Process,
				StartErmesCode:     pendingBefore.record.ErmesCode,
				EndErmesCode:       pt.record.ErmesCode,
				StartSRID:          pendingBefore.record.SRID,
				EndSRID:            pt.record.SRID,
				Details:            fmt.Sprintf("Rutina completada exitosamente (+%.1f°C en %ds)", rise, dur),
			})
			routineID++
			pendingBefore = nil
			continue
		}

		if pt.isWarmII {
			routines = append(routines, WarmupRoutineCycle{
				ID:                 routineID,
				RoutineType:        RoutineTypeWarmupII,
				Status:             "COMPLETED",
				StartTime:          pt.record.Date,
				StartTimeISO:       pt.record.TimestampISO,
				StartEpoch:         pt.record.Epoch,
				EndTime:            pt.record.Date,
				EndTimeISO:         pt.record.TimestampISO,
				EndEpoch:           pt.record.Epoch,
				DurationSeconds:    0,
				InitialTempCelsius: pt.temp,
				FinalTempCelsius:   pt.temp,
				TempRiseCelsius:    0,
				Process:            pt.record.Process,
				StartErmesCode:     pt.record.ErmesCode,
				StartSRID:          pt.record.SRID,
				Details:            fmt.Sprintf("Ciclo de calentamiento intermedio WarmupII registrado a %.1f°C", pt.temp),
			})
			routineID++
		}
	}

	// Si quedó el último before pendiente
	if pendingBefore != nil {
		routines = append(routines, WarmupRoutineCycle{
			ID:                 routineID,
			RoutineType:        RoutineTypeColdWarmup,
			Status:             "INCOMPLETE",
			StartTime:          pendingBefore.record.Date,
			StartTimeISO:       pendingBefore.record.TimestampISO,
			StartEpoch:         pendingBefore.record.Epoch,
			InitialTempCelsius: pendingBefore.temp,
			FinalTempCelsius:   pendingBefore.temp,
			TempRiseCelsius:    0,
			Process:            pendingBefore.record.Process,
			StartErmesCode:     pendingBefore.record.ErmesCode,
			StartSRID:          pendingBefore.record.SRID,
			Details:            fmt.Sprintf("Calentamiento en frío iniciado a %.1f°C", pendingBefore.temp),
		})
	}

	// Ordenar rutinas de más reciente a más antigua para visualización
	sort.Slice(routines, func(i, j int) bool {
		return routines[i].StartEpoch > routines[j].StartEpoch
	})

	// Ordenar skipped de más reciente a más antiguo
	sort.Slice(skippedList, func(i, j int) bool {
		return skippedList[i].Epoch > skippedList[j].Epoch
	})

	// Construir resumen estadístico
	summary := WarmupSummary{
		TotalCompletedRoutines: len(routines),
		TotalSkippedEvents:     len(skippedList),
		TubeThermalStatus:      "READY",
		ComplianceStatus:       "COMPLIANT",
	}

	if len(skippedList) > 0 {
		summary.ComplianceStatus = "ATTENTION_REQUIRED"
		lastSkip := skippedList[0]
		summary.LatestSkippedEvent = &lastSkip
	}

	var sumRise float64
	var sumDur int
	completedCount := 0

	for _, r := range routines {
		if r.Status == "COMPLETED" && r.TempRiseCelsius > 0 {
			sumRise += r.TempRiseCelsius
			sumDur += r.DurationSeconds
			completedCount++
		}
	}

	if completedCount > 0 {
		summary.AverageTempRiseCelsius = math.Round((sumRise/float64(completedCount))*100) / 100
		summary.AverageDurationSeconds = math.Round((float64(sumDur)/float64(completedCount))*10) / 10
	}

	if len(routines) > 0 {
		last := routines[0]
		summary.LastRoutineDate = last.StartTime
		summary.LastRoutineDateISO = last.StartTimeISO
		summary.LastInitialTempCelsius = last.InitialTempCelsius
		summary.LastFinalTempCelsius = last.FinalTempCelsius
		summary.LastTempRiseCelsius = last.TempRiseCelsius

		if last.FinalTempCelsius < 250.0 {
			summary.TubeThermalStatus = "NEEDS_WARMUP"
		} else if last.FinalTempCelsius > 2200.0 {
			summary.TubeThermalStatus = "OVERHEATED"
		} else {
			summary.TubeThermalStatus = "READY"
		}
	}

	return routines, skippedList, summary
}
