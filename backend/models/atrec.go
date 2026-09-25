package models

import "time"

// Constantes de ciclo de vida del Ticket ATREC
const (
	StatusOpen               = "OPEN"
	StatusInProgress         = "IN_PROGRESS"
	StatusPendingCalibration = "PENDING_CALIBRATION"
	StatusQAVerified         = "QA_VERIFIED"
	StatusClosed             = "CLOSED"
)

// Constantes de Severidad del Ticket ATREC
const (
	SeverityWarning  = "WARNING"  // Warning (Menor)
	SeverityMajor    = "MAJOR"    // Major (Mayor)
	SeverityCritical = "CRITICAL" // Critical (Crítica)
)

// Constantes de Tipo de Resolución Post-Reparación
const (
	ResolutionGeneralReview      = "REVISION_GENERAL"   // Revisión General
	ResolutionFieldDiagnosis     = "DIAGNOSTICO_CAMPO"  // Diagnóstico Técnico de Campo
	ResolutionTroubleshootingL1  = "TROUBLESHOOTING_L1" // Troubleshooting Nivel L1 Ejecutado
	ResolutionEscalationL2       = "ESCALADO_L2"        // Escalado a Nivel L2 (Especialista)
	ResolutionSparePartsRequired = "REQUIERE_REPUESTOS" // Requiere Repuestos
)

// PostRepairQA representa las pruebas de calibración y calidad de imagen post-reparación.
type PostRepairQA struct {
	ZAlignmentPassed            bool   `json:"z_alignment_passed"`
	PhantomIQPassed             bool   `json:"phantom_iq_passed"`
	DetectorFlatfieldCalibrated bool   `json:"detector_flatfield_calibrated"`
	QASignOffBy                 string `json:"qa_sign_off_by"`
	ResolutionType              string `json:"resolution_type,omitempty"` // Tipo de resolución técnica ejecutada
}

// TicketATREC representa el modelo de datos de la bitácora técnica ATREC.
type TicketATREC struct {
	TicketID           string       `json:"ticket_id"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
	TSMSubsystemID     int          `json:"tsm_subsystem_id"`
	Severity           string       `json:"severity"` // "WARNING" (Menor), "MAJOR" (Mayor), "CRITICAL" (Crítica)
	SourceErrorCode    string       `json:"source_error_code"`
	Description        string       `json:"description"`
	AssignedTechnician string       `json:"assigned_technician"`
	Status             string       `json:"status"` // "OPEN", "IN_PROGRESS", "PENDING_CALIBRATION", "QA_VERIFIED", "CLOSED"
	PostRepairQA       PostRepairQA `json:"post_repair_qa"`
}

// CreateTicketRequest payload esperado para registrar un nuevo ticket.
type CreateTicketRequest struct {
	TSMSubsystemID     int    `json:"tsm_subsystem_id"`
	Severity           string `json:"severity"` // "WARNING", "MAJOR", "CRITICAL"
	SourceErrorCode    string `json:"source_error_code"`
	Description        string `json:"description"`
	AssignedTechnician string `json:"assigned_technician"`
}

// CalibrationRequest payload esperado para registrar/actualizar pruebas de calidad de imagen.
type CalibrationRequest struct {
	ZAlignmentPassed            bool   `json:"z_alignment_passed"`
	PhantomIQPassed             bool   `json:"phantom_iq_passed"`
	DetectorFlatfieldCalibrated bool   `json:"detector_flatfield_calibrated"`
	QASignOffBy                 string `json:"qa_sign_off_by"`
	ResolutionType              string `json:"resolution_type"` // Revisión General, Diagnóstico Técnico de Campo, etc.
}
