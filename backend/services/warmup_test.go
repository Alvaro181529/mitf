package services

import (
	"testing"

	"mitfv2/internal/parser"
)

func TestParseWarmupRecords(t *testing.T) {
	sampleRecords := []parser.GELogRecord{
		{
			SRID:         "SR 187",
			Date:         "Thu Sep  3 17:28:45 2026",
			TimestampISO: "2026-09-03T17:28:45Z",
			Epoch:        1788456525,
			ErmesCode:    "230023009",
			Process:      "dailyPrepRx",
			Message:      "StateMachineEventNotify.c 499 Tube temperature before Cold Tube Warmup is 346.00 degrees Celsius.",
		},
		{
			SRID:         "SR 186",
			Date:         "Thu Sep  3 17:30:04 2026",
			TimestampISO: "2026-09-03T17:30:04Z",
			Epoch:        1788456604,
			ErmesCode:    "230023010",
			Process:      "dailyPrepRx",
			Message:      "StateMachineEventNotify.c 492 Tube temperature after Cold Tube Warmup is 612.29 degrees Celsius.",
		},
		{
			SRID:         "SR 265",
			Date:         "Sat Sep  5 08:18:47 2026",
			TimestampISO: "2026-09-05T08:18:47Z",
			Epoch:        1788596327,
			ErmesCode:    "200110044",
			Process:      "scanRx",
			Severity:     "Fatal",
			Message:      "Function: Data Acquisition : OC Processing Tube warm up was skipped. Limited mA will be used for exam 33283. Max mA for 120kV = 440, Max mA for 140kV= 380",
		},
		{
			SRID:         "SR 501",
			Date:         "Sun Sep  6 10:00:00 2026",
			TimestampISO: "2026-09-06T10:00:00Z",
			Epoch:        1788688800,
			ErmesCode:    "230023011",
			Process:      "dailyPrepRx",
			Message:      "Tube temperature before WarmupII is 619.78 degrees Celsius.",
		},
	}

	routines, skipped, summary := ParseWarmupRecords(sampleRecords)

	if len(routines) != 2 {
		t.Fatalf("Esperado 2 rutinas, obtenido %d", len(routines))
	}

	if len(skipped) != 1 {
		t.Fatalf("Esperado 1 skipped event, obtenido %d", len(skipped))
	}

	// Verificar el evento skipped
	if skipped[0].ExamID != "33283" {
		t.Errorf("ExamID esperado '33283', obtenido %q", skipped[0].ExamID)
	}
	if skipped[0].MaxMa120kV != 440 || skipped[0].MaxMa140kV != 380 {
		t.Errorf("Límites mA esperados (440, 380), obtenidos (%d, %d)", skipped[0].MaxMa120kV, skipped[0].MaxMa140kV)
	}

	// Verificar que la rutina cold tube warmup calculó correctamente la subida de temperatura y duración
	var coldRoutine *WarmupRoutineCycle
	for _, r := range routines {
		if r.RoutineType == RoutineTypeColdWarmup {
			coldRoutine = &r
			break
		}
	}

	if coldRoutine == nil {
		t.Fatal("No se encontró rutina COLD_TUBE_WARMUP")
	}

	if coldRoutine.InitialTempCelsius != 346.00 {
		t.Errorf("Temp inicial esperada 346.00, obtenida %.2f", coldRoutine.InitialTempCelsius)
	}
	if coldRoutine.FinalTempCelsius != 612.29 {
		t.Errorf("Temp final esperada 612.29, obtenida %.2f", coldRoutine.FinalTempCelsius)
	}
	if coldRoutine.TempRiseCelsius != 266.29 {
		t.Errorf("Temp rise esperado 266.29, obtenido %.2f", coldRoutine.TempRiseCelsius)
	}
	if coldRoutine.DurationSeconds != 79 {
		t.Errorf("Duración esperada 79s, obtenida %ds", coldRoutine.DurationSeconds)
	}

	// Verificar KPIs
	if summary.ComplianceStatus != "ATTENTION_REQUIRED" {
		t.Errorf("ComplianceStatus esperado 'ATTENTION_REQUIRED', obtenido %s", summary.ComplianceStatus)
	}
	if summary.TubeThermalStatus != "READY" {
		t.Errorf("TubeThermalStatus esperado 'READY', obtenido %s", summary.TubeThermalStatus)
	}
}
