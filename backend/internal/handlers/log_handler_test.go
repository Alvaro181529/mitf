package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mitfv2/config"
	"mitfv2/internal/parser"
	"mitfv2/internal/sshclient"
)

func TestHealthHandler(t *testing.T) {
	config.SSHHost = "192.168.1.50"
	config.SSHPort = 2222
	config.SSHUser = "admin"
	config.RemoteLogPath = "/opt/logs"

	client := sshclient.NewSSHClient()
	handler := NewLogHandler(client, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	handler.HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Código de estado esperado 200, obtenido %d", rr.Code)
	}

	var resp JSONResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error deserializando JSON: %v", err)
	}

	if !resp.Success {
		t.Errorf("Se esperaba success=true")
	}

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Data no es un mapa: %v", resp.Data)
	}

	if dataMap["ssh_host"] != "192.168.1.50" {
		t.Errorf("ssh_host esperado '192.168.1.50', obtenido %v", dataMap["ssh_host"])
	}
	if dataMap["remote_log_path"] != "/opt/logs" {
		t.Errorf("remote_log_path esperado '/opt/logs', obtenido %v", dataMap["remote_log_path"])
	}
}

func TestReadLogHandlerMissingFileParam(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewLogHandler(client, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/logs/read", nil)
	rr := httptest.NewRecorder()

	handler.ReadLogHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Código de estado esperado 400 por falta de parámetro, obtenido %d", rr.Code)
	}
}

func TestGetLogsHandlerNotConfigured(t *testing.T) {
	config.SSHPassword = ""
	config.SSHKeyPath = ""
	client := sshclient.NewSSHClient()
	handler := NewLogHandler(client, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/logs?severity=3&page=1&limit=20", nil)
	rr := httptest.NewRecorder()

	handler.GetLogsHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("Código esperado 500 al no tener credenciales SSH, obtenido %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error deserializando respuesta JSON: %v", err)
	}
	if resp["error"] == nil || resp["error"] == "" {
		t.Errorf("Se esperaba campo 'error' en respuesta con fallo")
	}
}

func TestParseRawTextHandlerWithFiltersAndPagination(t *testing.T) {
	client := sshclient.NewSSHClient()
	handler := NewLogHandler(client, nil)

	sample := `
SR 409
1788452500	0	2	Thu Sep  3 16:19:13 2026	260114056	2
	ct99	tgp
tgp_ctrl.cxx		112

Thu Sep  3 16:19:13 2026
Host : Table Gantry Processor    Ermes # : 260114056
Device : Gantry Assembly
[STOP SCAN] pushbutton on Console Push Button (GSCB) was pressed.

EN 409

SR 410
1788452600	0	1	Thu Sep  3 16:25:00 2026	260199001	1
	ct99	tubemgr
tube_diag.cxx		512

Thu Sep  3 16:25:00 2026
Host : Tube Manager subsystem    Ermes # : 260199001
Device : X-Ray Tube Assembly
Tube Health Index : 87.5%
Serial Number : SN-98234-CT
Tube thermal check completed successfully.

EN 410
`
	req := httptest.NewRequest(http.MethodPost, "/api/parser/ct99?host=tgp", strings.NewReader(sample))
	rr := httptest.NewRecorder()

	handler.ParseRawTextHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Código esperado 200, obtenido %d", rr.Code)
	}

	var resp parser.PaginatedLogsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error al deserializar JSON: %v", err)
	}

	if resp.Pagination.TotalRecords != 1 {
		t.Errorf("TotalRecords esperado 1, obtenido %d", resp.Pagination.TotalRecords)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Se esperaba 1 registro devuelto, obtenidos %d", len(resp.Data))
	}
	if resp.Data[0].Host != "Table Gantry Processor" {
		t.Errorf("Host esperado 'Table Gantry Processor', obtenido %q", resp.Data[0].Host)
	}
	if resp.Data[0].Process != "tgp" {
		t.Errorf("Process esperado 'tgp', obtenido %q", resp.Data[0].Process)
	}
}
