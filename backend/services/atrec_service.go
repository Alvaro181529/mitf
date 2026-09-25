package services

import (
	"fmt"
	"strings"

	"mitfv2/models"
)

// ValidationResult encapsula el resultado de la validación ATREC para cierre de ticket.
type ValidationResult struct {
	Valid         bool     `json:"valid"`
	MissingChecks []string `json:"missing_checks,omitempty"`
	ErrorMessage  string   `json:"error_message,omitempty"`
}

// ValidateTicketClosure ejecuta la regla de negocio crítica ATREC:
// Para cerrar un ticket (CLOSED), las 3 pruebas de PostRepairQA deben ser verdaderas:
// 1. ZAlignmentPassed (Alineación geométrica en Z)
// 2. PhantomIQPassed (Calidad de imagen con fantoma: ruido, resolución, HU)
// 3. DetectorFlatfieldCalibrated (Calibración Offset/Gain ADF)
//
// Retorna ValidationResult indicando si es válido y la lista detallada de pruebas pendientes.
func ValidateTicketClosure(qa models.PostRepairQA) ValidationResult {
	var missing []string

	if !qa.ZAlignmentPassed {
		missing = append(missing, "ZAlignmentPassed (Alineación geométrica en Z no aprobada)")
	}
	if !qa.PhantomIQPassed {
		missing = append(missing, "PhantomIQPassed (Prueba de calidad de imagen Phantom IQ no aprobada)")
	}
	if !qa.DetectorFlatfieldCalibrated {
		missing = append(missing, "DetectorFlatfieldCalibrated (Calibración Offset/Gain ADF de detector no aprobada)")
	}

	if len(missing) > 0 {
		return ValidationResult{
			Valid:         false,
			MissingChecks: missing,
			ErrorMessage:  fmt.Sprintf("Precondición fallida: El ticket no puede cerrarse sin registrar y aprobar todas las pruebas de calibración post-reparación. Pruebas faltantes: %s", strings.Join(missing, "; ")),
		}
	}

	return ValidationResult{
		Valid: true,
	}
}

// IsCalibrationFullyPassed verifica si las 3 pruebas de calibración fueron aprobadas.
func IsCalibrationFullyPassed(qa models.PostRepairQA) bool {
	return qa.ZAlignmentPassed && qa.PhantomIQPassed && qa.DetectorFlatfieldCalibrated
}
