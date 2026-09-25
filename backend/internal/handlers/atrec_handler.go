package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mitfv2/internal/database"
	"mitfv2/models"
	"mitfv2/services"
)

// ATRECHandler gestiona los endpoints REST de la bitácora técnica ATREC.
type ATRECHandler struct {
	repo database.ATRECRepository
}

// NewATRECHandler crea una nueva instancia inyectando el repositorio ATREC.
func NewATRECHandler(repo database.ATRECRepository) *ATRECHandler {
	return &ATRECHandler{
		repo: repo,
	}
}

// TicketsRouter despacha peticiones hacia /api/v1/atrec/tickets y sus sub-rutas según método HTTP.
// Rutas soportadas:
// - GET    /api/v1/atrec/tickets (?status=OPEN)
// - POST   /api/v1/atrec/tickets
// - PUT    /api/v1/atrec/tickets/:id/calibration
// - PATCH  /api/v1/atrec/tickets/:id/close
func (h *ATRECHandler) TicketsRouter(w http.ResponseWriter, r *http.Request) {
	// Limpiar prefijo base
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/atrec/tickets")
	path = strings.TrimPrefix(path, "/api/atrec/tickets")
	path = strings.Trim(path, "/")

	// 1. /api/v1/atrec/tickets (Colección)
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.ListTicketsHandler(w, r)
		case http.MethodPost:
			h.CreateTicketHandler(w, r)
		default:
			sendJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"status":  "error",
				"message": "Método no permitido. Use GET o POST.",
			})
		}
		return
	}

	// 2. Sub-rutas con ID: :id/calibration o :id/close
	parts := strings.Split(path, "/")
	if len(parts) == 2 {
		ticketID := parts[0]
		action := parts[1]

		switch action {
		case "calibration":
			if r.Method == http.MethodPut || r.Method == http.MethodPost {
				h.UpdateCalibrationHandler(w, r, ticketID)
				return
			}
			sendJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"status":  "error",
				"message": "Método no permitido. Use PUT para registrar calibración.",
			})
			return

		case "close":
			if r.Method == http.MethodPatch || r.Method == http.MethodPut || r.Method == http.MethodPost {
				h.CloseTicketHandler(w, r, ticketID)
				return
			}
			sendJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"status":  "error",
				"message": "Método no permitido. Use PATCH para cerrar el ticket.",
			})
			return

		default:
			sendJSON(w, http.StatusNotFound, map[string]interface{}{
				"status":  "error",
				"message": fmt.Sprintf("Acción '%s' no reconocida", action),
			})
			return
		}
	}

	// 3. /api/v1/atrec/tickets/:id
	if len(parts) == 1 {
		ticketID := parts[0]
		switch r.Method {
		case http.MethodGet:
			h.GetTicketByIDHandler(w, r, ticketID)
		default:
			sendJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"status":  "error",
				"message": "Método no permitido en ticket individual.",
			})
		}
		return
	}

	sendJSON(w, http.StatusNotFound, map[string]interface{}{
		"status":  "error",
		"message": "Ruta no encontrada.",
	})
}

// ListTicketsHandler lista todos los tickets con filtro opcional query: ?status=OPEN
// GET /api/v1/atrec/tickets
func (h *ATRECHandler) ListTicketsHandler(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")

	tickets, err := h.repo.GetAll(statusFilter)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Error al obtener tickets: %v", err),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"status_filter": statusFilter,
		"total_count":   len(tickets),
		"data":          tickets,
	})
}

// CreateTicketHandler registra un nuevo ticket de falla.
// POST /api/v1/atrec/tickets
func (h *ATRECHandler) CreateTicketHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "Error al leer el cuerpo de la petición",
		})
		return
	}
	defer r.Body.Close()

	var req models.CreateTicketRequest
	if err := json.Unmarshal(body, &req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("JSON inválido: %v", err),
		})
		return
	}

	if req.TSMSubsystemID < 1 || req.TSMSubsystemID > 5 {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "tsm_subsystem_id debe ser un entero entre 1 y 5",
		})
		return
	}

	if strings.TrimSpace(req.Description) == "" {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "El campo 'description' es requerido",
		})
		return
	}

	// Normalizar Severidad: WARNING, MAJOR, CRITICAL
	sev := strings.ToUpper(strings.TrimSpace(req.Severity))
	switch sev {
	case models.SeverityWarning, models.SeverityMajor, models.SeverityCritical:
		// Válido
	default:
		sev = models.SeverityWarning // Por defecto Warning (Menor)
	}

	now := time.Now().UTC()
	ticketID := h.repo.GenerateTicketID()

	newTicket := models.TicketATREC{
		TicketID:           ticketID,
		CreatedAt:          now,
		UpdatedAt:          now,
		TSMSubsystemID:     req.TSMSubsystemID,
		Severity:           sev,
		SourceErrorCode:    strings.TrimSpace(req.SourceErrorCode),
		Description:        strings.TrimSpace(req.Description),
		AssignedTechnician: strings.TrimSpace(req.AssignedTechnician),
		Status:             models.StatusOpen,
		PostRepairQA: models.PostRepairQA{
			ZAlignmentPassed:            false,
			PhantomIQPassed:             false,
			DetectorFlatfieldCalibrated: false,
			QASignOffBy:                 "",
			ResolutionType:              "",
		},
	}

	created, err := h.repo.Create(newTicket)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Error al almacenar el ticket: %v", err),
		})
		return
	}

	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Ticket ATREC creado exitosamente",
		"data":    created,
	})
}

// GetTicketByIDHandler obtiene un ticket específico por su ID.
// GET /api/v1/atrec/tickets/:id
func (h *ATRECHandler) GetTicketByIDHandler(w http.ResponseWriter, r *http.Request, ticketID string) {
	ticket, err := h.repo.GetByID(ticketID)
	if err != nil {
		sendJSON(w, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   ticket,
	})
}

// UpdateCalibrationHandler registra y actualiza las pruebas de calidad de imagen post-reparación.
// PUT /api/v1/atrec/tickets/:id/calibration
// Si las 3 pruebas son true, actualiza automáticamente el estado a "QA_VERIFIED".
func (h *ATRECHandler) UpdateCalibrationHandler(w http.ResponseWriter, r *http.Request, ticketID string) {
	ticket, err := h.repo.GetByID(ticketID)
	if err != nil {
		sendJSON(w, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "Error al leer el cuerpo de la petición",
		})
		return
	}
	defer r.Body.Close()

	var req models.CalibrationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("JSON inválido: %v", err),
		})
		return
	}

	// Actualizar PostRepairQA
	ticket.PostRepairQA.ZAlignmentPassed = req.ZAlignmentPassed
	ticket.PostRepairQA.PhantomIQPassed = req.PhantomIQPassed
	ticket.PostRepairQA.DetectorFlatfieldCalibrated = req.DetectorFlatfieldCalibrated
	if req.QASignOffBy != "" {
		ticket.PostRepairQA.QASignOffBy = strings.TrimSpace(req.QASignOffBy)
	}
	if req.ResolutionType != "" {
		ticket.PostRepairQA.ResolutionType = strings.TrimSpace(req.ResolutionType)
	}

	// Regla: si las 3 pruebas son true, actualizar automáticamente a "QA_VERIFIED"
	// a menos que ya esté en "CLOSED"
	if services.IsCalibrationFullyPassed(ticket.PostRepairQA) {
		if ticket.Status != models.StatusClosed {
			ticket.Status = models.StatusQAVerified
		}
	} else {
		// Si faltan pruebas y estaba en QA_VERIFIED, retrocede a PENDING_CALIBRATION
		if ticket.Status == models.StatusQAVerified {
			ticket.Status = models.StatusPendingCalibration
		}
	}

	updated, err := h.repo.Update(*ticket)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Error al actualizar calibración: %v", err),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":                   "success",
		"message":                  "Pruebas de calibración y calidad de imagen registradas exitosamente",
		"calibration_fully_passed": services.IsCalibrationFullyPassed(ticket.PostRepairQA),
		"data":                     updated,
	})
}

// CloseTicketHandler ejecuta la validación de negocio crítica ATREC antes de cambiar status a "CLOSED".
// PATCH /api/v1/atrec/tickets/:id/close
// Regla:
// - Retorna 200 OK si cumple las reglas ATREC (ZAlignmentPassed, PhantomIQPassed, DetectorFlatfieldCalibrated = true).
// - Retorna 412 Precondition Failed si falta alguna calibración, detallando los faltantes.
func (h *ATRECHandler) CloseTicketHandler(w http.ResponseWriter, r *http.Request, ticketID string) {
	ticket, err := h.repo.GetByID(ticketID)
	if err != nil {
		sendJSON(w, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// Ejecutar validación de negocio crítica ATREC
	valResult := services.ValidateTicketClosure(ticket.PostRepairQA)
	if !valResult.Valid {
		sendJSON(w, http.StatusPreconditionFailed, map[string]interface{}{
			"status":         "error",
			"code":           http.StatusPreconditionFailed,
			"error":          "Precondition Failed (Regla de Calibración ATREC no cumplida)",
			"message":        valResult.ErrorMessage,
			"missing_checks": valResult.MissingChecks,
			"ticket_id":      ticket.TicketID,
			"current_status": ticket.Status,
			"post_repair_qa": ticket.PostRepairQA,
		})
		return
	}

	// Validación superada: Actualizar estado a "CLOSED"
	ticket.Status = models.StatusClosed
	updated, err := h.repo.Update(*ticket)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Error al cerrar ticket: %v", err),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Ticket ATREC cerrado exitosamente. Equipo verificado y reincorporado al servicio clínico.",
		"data":    updated,
	})
}
