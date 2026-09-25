package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mitfv2/config"
	"mitfv2/internal/sshclient"
	"mitfv2/services"
)

func init() {
	config.LoadConfig()
}

func TestHardwareSummaryHandler(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewHardwareHandler(client, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hardware/summary", nil)
	rr := httptest.NewRecorder()

	handler.HardwareSummaryHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Código esperado 200, obtenido %d", rr.Code)
	}

	var resp services.HardwareSummaryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error al deserializar JSON: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Status esperado 'success', obtenido %q", resp.Status)
	}
	if resp.Data.TubeRX.LimitNominalMas != 100000000 {
		t.Errorf("LimitNominalMas esperado 100000000, obtenido %d", resp.Data.TubeRX.LimitNominalMas)
	}
	if resp.Data.GantryRotor.LimitRevolutions != 100000 {
		t.Errorf("LimitRevolutions esperado 100000, obtenido %d", resp.Data.GantryRotor.LimitRevolutions)
	}
	if len(resp.Data.TSMSubsystemsHealth) != 5 {
		t.Errorf("Cantidad esperada de subsistemas TSM: 5, obtenido %d", len(resp.Data.TSMSubsystemsHealth))
	}
}

func TestTubeHealthHandler(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewHardwareHandler(client, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hardware/tube-health", nil)
	rr := httptest.NewRecorder()

	handler.TubeHealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Código esperado 200, obtenido %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error al deserializar JSON: %v", err)
	}

	if resp["status"] != "success" {
		t.Errorf("Status esperado 'success', obtenido %v", resp["status"])
	}

	dataMap, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Data no es un mapa")
	}

	limitMas, _ := dataMap["limit_nominal_mas"].(float64)
	if int64(limitMas) != 100000000 {
		t.Errorf("limit_nominal_mas esperado 100000000, obtenido %v", limitMas)
	}
}

func TestTubeWarmupHandler(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewHardwareHandler(client, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hardware/tube-warmup?limit=10", nil)
	rr := httptest.NewRecorder()

	handler.TubeWarmupHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Código esperado 200, obtenido %d (Body: %s)", rr.Code, rr.Body.String())
	}

	var resp services.TubeWarmupResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error al deserializar JSON: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Status esperado 'success', obtenido %q", resp.Status)
	}

	if resp.Summary.TotalCompletedRoutines <= 0 {
		t.Errorf("Se esperaban rutinas de calentamiento registradas en gesys_ct99.log, obtenido %d", resp.Summary.TotalCompletedRoutines)
	}

	if len(resp.Routines) == 0 {
		t.Errorf("La lista de rutinas no debería estar vacía")
	}
}

func TestGantryStatsHandler(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewHardwareHandler(client, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hardware/gantry-stats", nil)
	rr := httptest.NewRecorder()

	handler.GantryStatsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Código esperado 200, obtenido %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error al deserializar JSON: %v", err)
	}

	dataMap, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Data no es un mapa")
	}

	limitRev, _ := dataMap["limit_revolutions"].(float64)
	if int64(limitRev) != 100000 {
		t.Errorf("limit_revolutions esperado 100000, obtenido %v", limitRev)
	}
}

func TestInteractiveMapHandler(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewHardwareHandler(client, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hardware/interactive-map", nil)
	rr := httptest.NewRecorder()

	handler.InteractiveMapHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("InteractiveMapHandler código esperado 200, obtenido %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error al deserializar JSON: %v", err)
	}

	if resp["system_status"] == nil || resp["system_status"] == "" {
		t.Errorf("system_status requerido en respuesta")
	}

	components, ok := resp["components"].([]interface{})
	if !ok || len(components) != 5 {
		t.Fatalf("Se esperaban 5 componentes en 'components', obtenidos %v", len(components))
	}

	firstComp := components[0].(map[string]interface{})
	if firstComp["node_id"] != "node_tube_xray" {
		t.Errorf("Primer nodo esperado 'node_tube_xray', obtenido %v", firstComp["node_id"])
	}
	if firstComp["color_hex"] == nil || firstComp["color_hex"] == "" {
		t.Errorf("color_hex esperado en componente")
	}
}

func TestTSMAlertsBySubsystemHandler(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewHardwareHandler(client, nil)

	// Subsystem 1 (Generador y Tubo RX)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/tsm/1", nil)
	rr := httptest.NewRecorder()

	handler.TSMAlertsBySubsystemHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Código esperado 200, obtenido %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error al deserializar JSON: %v", err)
	}

	if resp["subsystem_name"] != "Generador y Tubo RX" {
		t.Errorf("Nombre de subsistema esperado 'Generador y Tubo RX', obtenido %v", resp["subsystem_name"])
	}

	// Subsystem ID inválido (ej: 99)
	reqErr := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/tsm/99", nil)
	rrErr := httptest.NewRecorder()

	handler.TSMAlertsBySubsystemHandler(rrErr, reqErr)

	if rrErr.Code != http.StatusBadRequest {
		t.Fatalf("Código esperado 400 para subsistema 99, obtenido %d", rrErr.Code)
	}
}

func TestTelemetryLogsEndpoints(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewHardwareHandler(client, nil)

	reqCan := httptest.NewRequest(http.MethodGet, "/api/v1/logs/jedi-can", nil)
	rrCan := httptest.NewRecorder()
	handler.JediCanLogsHandler(rrCan, reqCan)
	if rrCan.Code != http.StatusOK && rrCan.Code != http.StatusInternalServerError {
		t.Fatalf("JediCanLogsHandler código inesperado %d", rrCan.Code)
	}

	reqDas := httptest.NewRequest(http.MethodGet, "/api/v1/logs/das-errors", nil)
	rrDas := httptest.NewRecorder()
	handler.DasErrorsLogsHandler(rrDas, reqDas)
	if rrDas.Code != http.StatusOK && rrDas.Code != http.StatusInternalServerError {
		t.Fatalf("DasErrorsLogsHandler código inesperado %d", rrDas.Code)
	}
}
