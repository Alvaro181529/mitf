package parser

import (
	"testing"
)

func TestParseAndFormatTimestamp(t *testing.T) {
	tests := []struct {
		name          string
		inputRaw      string
		inputEpoch    int64
		wantFormatted string
		wantISO       string
		wantRaw       string
	}{
		{
			name:          "User timestamp Mon Sep 14",
			inputRaw:      "Mon Sep 14 17:11:27 2026",
			inputEpoch:    0,
			wantFormatted: "2026-09-14 17:11:27",
			wantISO:       "2026-09-14T17:11:27Z",
			wantRaw:       "Mon Sep 14 17:11:27 2026",
		},
		{
			name:          "User timestamp single digit day Thu Sep 3",
			inputRaw:      "Thu Sep  3 16:25:00 2026",
			inputEpoch:    1788452500,
			wantFormatted: "2026-09-03 16:25:00",
			wantISO:       "2026-09-03T16:25:00Z",
			wantRaw:       "Thu Sep  3 16:25:00 2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted, iso, raw, epoch := ParseAndFormatTimestamp(tt.inputRaw, tt.inputEpoch)
			if formatted != tt.wantFormatted {
				t.Errorf("formatted = %q, want %q", formatted, tt.wantFormatted)
			}
			if iso != tt.wantISO {
				t.Errorf("iso = %q, want %q", iso, tt.wantISO)
			}
			if raw != tt.wantRaw {
				t.Errorf("raw = %q, want %q", raw, tt.wantRaw)
			}
			if epoch <= 0 {
				t.Errorf("epoch debe ser positivo, obtenido %d", epoch)
			}
		})
	}
}

func TestParseCT99LogBlock(t *testing.T) {
	sample := `
SR 812
1788452500	0	1	Mon Sep 14 17:11:27 2026	260199001	1
	ct99	tubemgr
tube_diag.cxx		512

Mon Sep 14 17:11:27 2026
Host : Tube Manager subsystem    Ermes # : 260199001
Device : X-Ray Tube Assembly
Tube Health Index : 87.5%
Serial Number : SN-98234-CT
Accumulated Scans : 14205
Total Slice Count : 284100
Tube Heat Load : 42%

Tube thermal check completed successfully.

EN 812
`

	records, err := ParseCT99String(sample, 0)
	if err != nil {
		t.Fatalf("Error inesperado al parsear: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Se esperaba 1 registro, obtenidos %d", len(records))
	}

	r := records[0]

	// 1. Metadatos y Timestamp mejorado
	if r.SRID != "SR 812" {
		t.Errorf("SR_ID esperado 'SR 812', obtenido %q", r.SRID)
	}
	if r.Date != "Mon Sep 14 17:11:27 2026" {
		t.Errorf("Date esperado 'Mon Sep 14 17:11:27 2026', obtenido %q", r.Date)
	}
	if r.Timestamp != "2026-09-14 17:11:27" {
		t.Errorf("Timestamp estándar esperado '2026-09-14 17:11:27', obtenido %q", r.Timestamp)
	}
	if r.TimestampISO != "2026-09-14T17:11:27Z" {
		t.Errorf("TimestampISO esperado '2026-09-14T17:11:27Z', obtenido %q", r.TimestampISO)
	}
	if r.TimestampRaw != "Mon Sep 14 17:11:27 2026" {
		t.Errorf("TimestampRaw esperado 'Mon Sep 14 17:11:27 2026', obtenido %q", r.TimestampRaw)
	}
	if r.ErmesCode != "260199001" {
		t.Errorf("ErmesCode esperado '260199001', obtenido %q", r.ErmesCode)
	}
	if r.SeverityCode != 1 || r.Severity != "Info" {
		t.Errorf("Severity esperado 1 (Info), obtenido %d (%s)", r.SeverityCode, r.Severity)
	}
	if r.Host != "Tube Manager subsystem" {
		t.Errorf("Host esperado 'Tube Manager subsystem', obtenido %q", r.Host)
	}
	if r.Process != "tubemgr" {
		t.Errorf("Process esperado 'tubemgr', obtenido %q", r.Process)
	}
	if r.SourceFile != "tube_diag.cxx:512" {
		t.Errorf("SourceFile esperado 'tube_diag.cxx:512', obtenido %q", r.SourceFile)
	}

	// 2. Atributos de Hardware y Diagnóstico
	if r.Device != "X-Ray Tube Assembly" {
		t.Errorf("Device esperado 'X-Ray Tube Assembly', obtenido %q", r.Device)
	}
	if r.SerialNumber != "SN-98234-CT" {
		t.Errorf("SerialNumber esperado 'SN-98234-CT', obtenido %q", r.SerialNumber)
	}
	if r.TubeHealthIndex != "87.5%" {
		t.Errorf("TubeHealthIndex esperado '87.5%%', obtenido %q", r.TubeHealthIndex)
	}
	if r.TubeHeatLoad != "42%" {
		t.Errorf("TubeHeatLoad esperado '42%%', obtenido %q", r.TubeHeatLoad)
	}
	if r.AccumulatedScans != "14205" {
		t.Errorf("AccumulatedScans esperado '14205', obtenido %q", r.AccumulatedScans)
	}
	if r.TotalSliceCount != "284100" {
		t.Errorf("TotalSliceCount esperado '284100', obtenido %q", r.TotalSliceCount)
	}
	if r.Message != "Tube thermal check completed successfully." {
		t.Errorf("Message esperado 'Tube thermal check completed successfully.', obtenido %q", r.Message)
	}
}

func TestParseMultipleBlocksAndOptionalFields(t *testing.T) {
	multi := `
SR 101
1788452500	0	2	Fri Sep  4 10:00:00 2026	260199002	2
	ct99	generator
gen_ctrl.cxx		120

Device : High Voltage Generator
Generator warning: oil temp high.

EN 101

SR 102
1788452600	0	3	Fri Sep  4 10:05:00 2026	260199003	3
	ct99	detector
das_main.cxx		340

reports a total of 15200.5 ma*s
Calibration failed due to packet loss.

EN 102
`
	records, err := ParseCT99String(multi, 0)
	if err != nil {
		t.Fatalf("Error inesperado: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Se esperaban 2 registros, obtenidos %d", len(records))
	}

	if records[0].SRID != "SR 101" || records[0].Severity != "Pri/Soft" || records[0].Device != "High Voltage Generator" {
		t.Errorf("Registro 1 falló validación: %+v", records[0])
	}
	if records[0].Timestamp != "2026-09-04 10:00:00" {
		t.Errorf("Timestamp registro 1 = %q", records[0].Timestamp)
	}

	if records[1].SRID != "SR 102" || records[1].Severity != "Pri/Hard" || records[1].MasAccumulated != "15200.5" {
		t.Errorf("Registro 2 falló validación: %+v", records[1])
	}
}
