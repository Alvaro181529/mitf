package services

import (
	"testing"
)

func TestCalculateTubeEOL(t *testing.T) {
	tests := []struct {
		mas  int64
		want float64
	}{
		{0, 0.0},
		{7478990, 7.48},
		{100000000, 100.0},
		{50000000, 50.0},
	}

	for _, tt := range tests {
		got := CalculateTubeEOL(tt.mas)
		if got != tt.want {
			t.Errorf("CalculateTubeEOL(%d) = %v, want %v", tt.mas, got, tt.want)
		}
	}
}

func TestCalculateRotorLifecycle(t *testing.T) {
	tests := []struct {
		revs int64
		want float64
	}{
		{0, 0.0},
		{45210, 45.21},
		{100000, 100.0},
		{75000, 75.0},
	}

	for _, tt := range tests {
		got := CalculateRotorLifecycle(tt.revs)
		if got != tt.want {
			t.Errorf("CalculateRotorLifecycle(%d) = %v, want %v", tt.revs, got, tt.want)
		}
	}
}

func TestEvaluateAnodeThermalRisk(t *testing.T) {
	tests := []struct {
		tempCelsius float64
		currentMHU  float64
		want        string
	}{
		{1850.0, 3.0, "NORMAL"},
		{2300.0, 4.0, "WARNING"},
		{2100.0, 5.5, "WARNING"},
		{2700.0, 3.0, "CRITICAL"},
		{1800.0, 7.0, "CRITICAL"},
	}

	for _, tt := range tests {
		got := EvaluateAnodeThermalRisk(tt.tempCelsius, tt.currentMHU)
		if got != tt.want {
			t.Errorf("EvaluateAnodeThermalRisk(%v, %v) = %q, want %q", tt.tempCelsius, tt.currentMHU, got, tt.want)
		}
	}
}

func TestFilterTSMAlerts(t *testing.T) {
	entries := []LogEntry{
		{Process: "tubemgr", Message: "Tube thermal check", SeverityCode: 1},
		{Process: "rotmgr", Message: "Rotor speed deviation", SeverityCode: 2},
		{Process: "tgp", Message: "Table movement error", SeverityCode: 3},
		{Process: "dasmgr", Message: "DAS channel out of range", SeverityCode: 2},
		{Process: "gscb", Message: "[STOP SCAN] pressed", SeverityCode: 2},
	}

	// Subsistema 1: Generador y Tubo RX
	sub1 := FilterTSMAlerts(entries, 1)
	if len(sub1) != 1 || sub1[0].Process != "tubemgr" {
		t.Errorf("Filtro subsistema 1 falló: %+v", sub1)
	}

	// Subsistema 2: Gantry y Rotor
	sub2 := FilterTSMAlerts(entries, 2)
	if len(sub2) != 1 || sub2[0].Process != "rotmgr" {
		t.Errorf("Filtro subsistema 2 falló: %+v", sub2)
	}

	// Subsistema 3: Mesa
	sub3 := FilterTSMAlerts(entries, 3)
	if len(sub3) != 1 || sub3[0].Process != "tgp" {
		t.Errorf("Filtro subsistema 3 falló: %+v", sub3)
	}

	// Subsistema 4: DAS
	sub4 := FilterTSMAlerts(entries, 4)
	if len(sub4) != 1 || sub4[0].Process != "dasmgr" {
		t.Errorf("Filtro subsistema 4 falló: %+v", sub4)
	}

	// Subsistema 5: Consola y Emergencia
	sub5 := FilterTSMAlerts(entries, 5)
	if len(sub5) != 1 || sub5[0].Process != "gscb" {
		t.Errorf("Filtro subsistema 5 falló: %+v", sub5)
	}
}

func TestDetectEmergencyStop(t *testing.T) {
	if !DetectEmergencyStop("260114056", "tgp") {
		t.Errorf("DetectEmergencyStop con código 260114056 debió ser true")
	}
	if !DetectEmergencyStop("0", "gscb") {
		t.Errorf("DetectEmergencyStop con proceso gscb debió ser true")
	}
	if !DetectEmergencyStop("STOP SCAN pressed", "other") {
		t.Errorf("DetectEmergencyStop con texto STOP SCAN debió ser true")
	}
	if DetectEmergencyStop("Normal scan", "tubemgr") {
		t.Errorf("DetectEmergencyStop con scan normal debió ser false")
	}
}

func TestFormatIntegerWithCommas(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{9, "9"},
		{123, "123"},
		{1000, "1,000"},
		{45210, "45,210"},
		{7478990, "7,478,990"},
		{173523251, "173,523,251"},
		{-1000, "-1,000"},
	}

	for _, tt := range tests {
		got := FormatIntegerWithCommas(tt.input)
		if got != tt.want {
			t.Errorf("FormatIntegerWithCommas(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildInteractiveMap(t *testing.T) {
	entries := []LogEntry{
		{Process: "tubemgr", Message: "Normal tube check", SeverityCode: 1},
		{Process: "rotmgr", Message: "Slip ring jitter threshold exceeded", SeverityCode: 2},
		{Process: "gscb", Message: "[STOP SCAN] pushbutton pressed", SeverityCode: 2},
	}

	tube := TubeHealthData{
		AccumulatedMas:          7478990,
		LimitNominalMas:         100000000,
		UsagePercentage:         7.48,
		AnodeTemperatureCelsius: 1850.0,
		AnodeThermalCapacityMHU: 5.34,
		Status:                  "NORMAL",
	}

	gantry := GantryStatsData{
		AccumulatedRevolutions: 45210,
		LimitRevolutions:       100000,
		UsagePercentage:        45.21,
		Status:                 "OPERATIONAL",
	}

	sysStatus, components := BuildInteractiveMap(entries, tube, gantry)

	if len(components) != 5 {
		t.Fatalf("Se esperaban 5 componentes en el mapa, obtenidos %d", len(components))
	}

	expectedNodes := []string{"node_tube_xray", "node_gantry_rotor", "node_couch_table", "node_das_detector", "node_estop_console"}
	for i, exp := range expectedNodes {
		if components[i].NodeID != exp {
			t.Errorf("Componente [%d] NodeID esperado %q, obtenido %q", i, exp, components[i].NodeID)
		}
		if components[i].TSMID != i+1 {
			t.Errorf("Componente [%d] TSMID esperado %d, obtenido %d", i, i+1, components[i].TSMID)
		}
	}

	if components[0].Status != "OK" || components[0].ColorHex != ColorStatusOK {
		t.Errorf("Tubo esperado OK (%s), obtenido %s (%s)", ColorStatusOK, components[0].Status, components[0].ColorHex)
	}
	if components[0].Metrics["mAs_acumulados"] != "7,478,990 (7.5%)" {
		t.Errorf("mAs_acumulados inesperado: %s", components[0].Metrics["mAs_acumulados"])
	}

	if components[1].Status != "WARNING" || components[1].ColorHex != ColorStatusWarning {
		t.Errorf("Gantry esperado WARNING (%s), obtenido %s (%s)", ColorStatusWarning, components[1].Status, components[1].ColorHex)
	}

	if components[4].Status != "CRITICAL" || components[4].ColorHex != ColorStatusCritical {
		t.Errorf("Consola esperada CRITICAL (%s), obtenido %s (%s)", ColorStatusCritical, components[4].Status, components[4].ColorHex)
	}

	if sysStatus != "CRITICAL" {
		t.Errorf("system_status global esperado CRITICAL, obtenido %q", sysStatus)
	}
}
