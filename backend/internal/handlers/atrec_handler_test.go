package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mitfv2/internal/database"
	"mitfv2/models"
)

func setupTestATRECHandler() (*ATRECHandler, *database.MemoryATRECRepository) {
	repo := database.NewMemoryATRECRepository()
	handler := NewATRECHandler(repo)
	return handler, repo
}

func TestListTicketsHandler(t *testing.T) {
	handler, _ := setupTestATRECHandler()

	// 1. Listar todos
	req := httptest.NewRequest(http.MethodGet, "/api/v1/atrec/tickets", nil)
	rr := httptest.NewRecorder()
	handler.TicketsRouter(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Código esperado 200, obtenido %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error deserializando JSON: %v", err)
	}

	totalCount := resp["total_count"].(float64)
	if totalCount < 2 {
		t.Errorf("Esperados al menos 2 tickets iniciales, obtenidos %v", totalCount)
	}

	// 2. Filtrar por ?status=OPEN
	reqFilter := httptest.NewRequest(http.MethodGet, "/api/v1/atrec/tickets?status=OPEN", nil)
	rrFilter := httptest.NewRecorder()
	handler.TicketsRouter(rrFilter, reqFilter)

	if rrFilter.Code != http.StatusOK {
		t.Fatalf("Código esperado 200, obtenido %d", rrFilter.Code)
	}

	var respFilter map[string]interface{}
	if err := json.Unmarshal(rrFilter.Body.Bytes(), &respFilter); err != nil {
		t.Fatalf("Error deserializando JSON: %v", err)
	}

	dataList := respFilter["data"].([]interface{})
	for _, item := range dataList {
		ticketMap := item.(map[string]interface{})
		if ticketMap["status"] != "OPEN" {
			t.Errorf("Ticket retornado con status distinto a OPEN: %v", ticketMap["status"])
		}
	}
}

func TestCreateTicketHandler(t *testing.T) {
	handler, _ := setupTestATRECHandler()

	payload := []byte(`{
		"tsm_subsystem_id": 1,
		"source_error_code": "260199001",
		"description": "Falla térmica en tubo de rayos X",
		"assigned_technician": "Ing. Martín Gómez"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/atrec/tickets", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.TicketsRouter(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Código esperado 201 Created, obtenido %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Error unmarshaling JSON: %v", err)
	}

	ticketData, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Data no es un objeto")
	}

	if ticketData["status"] != models.StatusOpen {
		t.Errorf("Estado inicial esperado 'OPEN', obtenido %v", ticketData["status"])
	}

	if ticketData["ticket_id"] == nil || ticketData["ticket_id"] == "" {
		t.Errorf("ticket_id no fue generado")
	}
}

func TestATRECValidationRuleAndClose(t *testing.T) {
	handler, repo := setupTestATRECHandler()

	// Crear ticket de prueba
	ticket := models.TicketATREC{
		TicketID:           "ATREC-2026-TEST",
		TSMSubsystemID:     1,
		SourceErrorCode:    "260199001",
		Description:        "Prueba de regla de negocio ATREC",
		AssignedTechnician: "Ing. Test",
		Status:             models.StatusOpen,
		PostRepairQA: models.PostRepairQA{
			ZAlignmentPassed:            false,
			PhantomIQPassed:             false,
			DetectorFlatfieldCalibrated: false,
		},
	}
	_, _ = repo.Create(ticket)

	// PASO 1: Intentar cerrar sin calibración -> Debe rechazar con HTTP 412 (Precondition Failed)
	reqClose1 := httptest.NewRequest(http.MethodPatch, "/api/v1/atrec/tickets/ATREC-2026-TEST/close", nil)
	rrClose1 := httptest.NewRecorder()
	handler.TicketsRouter(rrClose1, reqClose1)

	if rrClose1.Code != http.StatusPreconditionFailed {
		t.Fatalf("Regla ATREC falló: se esperaba HTTP 412 Precondition Failed, se obtuvo %d: %s", rrClose1.Code, rrClose1.Body.String())
	}

	var resp412 map[string]interface{}
	if err := json.Unmarshal(rrClose1.Body.Bytes(), &resp412); err != nil {
		t.Fatalf("Error unmarshaling JSON: %v", err)
	}

	missingList, ok := resp412["missing_checks"].([]interface{})
	if !ok || len(missingList) != 3 {
		t.Errorf("Se esperaban 3 pruebas faltantes detalladas en 412, obtenidas %v", len(missingList))
	}

	// PASO 2: Calibración parcial (solo 2 de 3)
	partialCalib := []byte(`{
		"z_alignment_passed": true,
		"phantom_iq_passed": true,
		"detector_flatfield_calibrated": false,
		"qa_sign_off_by": "Tec. QA"
	}`)
	reqCalibPart := httptest.NewRequest(http.MethodPut, "/api/v1/atrec/tickets/ATREC-2026-TEST/calibration", bytes.NewBuffer(partialCalib))
	rrCalibPart := httptest.NewRecorder()
	handler.TicketsRouter(rrCalibPart, reqCalibPart)

	if rrCalibPart.Code != http.StatusOK {
		t.Fatalf("Código esperado 200 en calibración, obtenido %d", rrCalibPart.Code)
	}

	// Intentar cerrar nuevamente -> Aún debe fallar con 412 (falta flatfield)
	reqClose2 := httptest.NewRequest(http.MethodPatch, "/api/v1/atrec/tickets/ATREC-2026-TEST/close", nil)
	rrClose2 := httptest.NewRecorder()
	handler.TicketsRouter(rrClose2, reqClose2)

	if rrClose2.Code != http.StatusPreconditionFailed {
		t.Fatalf("Se esperaba 412 con calibración parcial, obtenido %d", rrClose2.Code)
	}

	// PASO 3: Registrar todas las 3 pruebas exitosas (Z, Phantom IQ, Flatfield)
	fullCalib := []byte(`{
		"z_alignment_passed": true,
		"phantom_iq_passed": true,
		"detector_flatfield_calibrated": true,
		"qa_sign_off_by": "Tec. QA Certificado"
	}`)
	reqCalibFull := httptest.NewRequest(http.MethodPut, "/api/v1/atrec/tickets/ATREC-2026-TEST/calibration", bytes.NewBuffer(fullCalib))
	rrCalibFull := httptest.NewRecorder()
	handler.TicketsRouter(rrCalibFull, reqCalibFull)

	if rrCalibFull.Code != http.StatusOK {
		t.Fatalf("Código esperado 200 en calibración completa, obtenido %d", rrCalibFull.Code)
	}

	// Verificar que el estado cambió automáticamente a QA_VERIFIED
	ticketVerif, _ := repo.GetByID("ATREC-2026-TEST")
	if ticketVerif.Status != models.StatusQAVerified {
		t.Errorf("Estado esperado 'QA_VERIFIED' tras calibración completa, obtenido '%s'", ticketVerif.Status)
	}

	// PASO 4: Cerrar ticket con calibración aprobada -> Debe responder 200 OK
	reqClose3 := httptest.NewRequest(http.MethodPatch, "/api/v1/atrec/tickets/ATREC-2026-TEST/close", nil)
	rrClose3 := httptest.NewRecorder()
	handler.TicketsRouter(rrClose3, reqClose3)

	if rrClose3.Code != http.StatusOK {
		t.Fatalf("Cierre de ticket con calibración aprobada debió ser 200 OK, obtenido %d: %s", rrClose3.Code, rrClose3.Body.String())
	}

	ticketClosed, _ := repo.GetByID("ATREC-2026-TEST")
	if ticketClosed.Status != models.StatusClosed {
		t.Errorf("Estado esperado 'CLOSED' tras cierre exitoso, obtenido '%s'", ticketClosed.Status)
	}
}
