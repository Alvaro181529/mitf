package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"mitfv2/config"
	"mitfv2/internal/database"
)

func init() {
	config.LoadConfig()
}

func TestConfigHandlerEndpoints(t *testing.T) {
	repo, err := database.InitDB()
	if err != nil {
		t.Skipf("Omitiendo pruebas HTTP de base de datos si PostgreSQL no está disponible: %v", err)
	}

	handler := NewConfigHandler(repo)

	// 1. GET /api/v1/configs
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/configs", nil)
	rrList := httptest.NewRecorder()
	handler.ConfigsRouter(rrList, reqList)

	if rrList.Code != http.StatusOK {
		t.Fatalf("ListConfigs esperado 200, obtenido %d", rrList.Code)
	}

	var listResp map[string]interface{}
	if err := json.Unmarshal(rrList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Error deserializando respuesta: %v", err)
	}
	if listResp["status"] != "success" {
		t.Errorf("Status esperado 'success', obtenido %v", listResp["status"])
	}

	// 2. GET /api/v1/configs/active
	reqActive := httptest.NewRequest(http.MethodGet, "/api/v1/configs/active", nil)
	rrActive := httptest.NewRecorder()
	handler.ConfigsRouter(rrActive, reqActive)

	if rrActive.Code != http.StatusOK {
		t.Fatalf("GetActiveConfig esperado 200, obtenido %d", rrActive.Code)
	}

	// 3. POST /api/v1/configs (Crear)
	newPayload := map[string]interface{}{
		"name":            "Servidor Tomógrafo Secundario",
		"ssh_host":        "192.168.122.99",
		"ssh_port":        22,
		"ssh_user":        "ct_user",
		"ssh_password":    "ct_pass",
		"remote_log_path": "/var/log/ct99",
		"server_port":     ":8080",
		"is_active":       false,
	}
	bodyBytes, _ := json.Marshal(newPayload)
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/configs", bytes.NewReader(bodyBytes))
	rrCreate := httptest.NewRecorder()
	handler.ConfigsRouter(rrCreate, reqCreate)

	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("CreateConfig esperado 201, obtenido %d: %s", rrCreate.Code, rrCreate.Body.String())
	}

	var createResp map[string]interface{}
	json.Unmarshal(rrCreate.Body.Bytes(), &createResp)
	createdData, ok := createResp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Respuesta data debe ser un mapa")
	}
	createdID := int64(createdData["id"].(float64))

	// 4. GET /api/v1/configs/{id}
	reqGetID := httptest.NewRequest(http.MethodGet, "/api/v1/configs/"+strconv.FormatInt(createdID, 10), nil)
	rrGetID := httptest.NewRecorder()
	handler.ConfigsRouter(rrGetID, reqGetID)

	if rrGetID.Code != http.StatusOK {
		t.Fatalf("GetConfigByID esperado 200, obtenido %d", rrGetID.Code)
	}

	// 5. PUT /api/v1/configs/{id}
	updatePayload := map[string]interface{}{
		"name": "Servidor Tomógrafo Editado",
	}
	upBytes, _ := json.Marshal(updatePayload)
	reqPut := httptest.NewRequest(http.MethodPut, "/api/v1/configs/"+strconv.FormatInt(createdID, 10), bytes.NewReader(upBytes))
	rrPut := httptest.NewRecorder()
	handler.ConfigsRouter(rrPut, reqPut)

	if rrPut.Code != http.StatusOK {
		t.Fatalf("UpdateConfig esperado 200, obtenido %d", rrPut.Code)
	}

	// 6. POST /api/v1/configs/{id}/activate
	reqAct := httptest.NewRequest(http.MethodPost, "/api/v1/configs/"+strconv.FormatInt(createdID, 10)+"/activate", nil)
	rrAct := httptest.NewRecorder()
	handler.ConfigsRouter(rrAct, reqAct)

	if rrAct.Code != http.StatusOK {
		t.Fatalf("ActivateConfig esperado 200, obtenido %d", rrAct.Code)
	}

	// 7. DELETE /api/v1/configs/{id}
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/configs/"+strconv.FormatInt(createdID, 10), nil)
	rrDel := httptest.NewRecorder()
	handler.ConfigsRouter(rrDel, reqDel)

	if rrDel.Code != http.StatusOK {
		t.Fatalf("DeleteConfig esperado 200, obtenido %d: %s", rrDel.Code, rrDel.Body.String())
	}
}
