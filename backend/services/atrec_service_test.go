package services

import (
	"testing"

	"mitfv2/models"
)

func TestValidateTicketClosure(t *testing.T) {
	tests := []struct {
		name          string
		qa            models.PostRepairQA
		wantValid     bool
		expectedFails int
	}{
		{
			name: "Todas las pruebas aprobadas",
			qa: models.PostRepairQA{
				ZAlignmentPassed:            true,
				PhantomIQPassed:             true,
				DetectorFlatfieldCalibrated: true,
				QASignOffBy:                 "Ing. Roberto Díaz",
			},
			wantValid:     true,
			expectedFails: 0,
		},
		{
			name: "Falta ZAlignmentPassed",
			qa: models.PostRepairQA{
				ZAlignmentPassed:            false,
				PhantomIQPassed:             true,
				DetectorFlatfieldCalibrated: true,
			},
			wantValid:     false,
			expectedFails: 1,
		},
		{
			name: "Falta PhantomIQPassed",
			qa: models.PostRepairQA{
				ZAlignmentPassed:            true,
				PhantomIQPassed:             false,
				DetectorFlatfieldCalibrated: true,
			},
			wantValid:     false,
			expectedFails: 1,
		},
		{
			name: "Falta DetectorFlatfieldCalibrated",
			qa: models.PostRepairQA{
				ZAlignmentPassed:            true,
				PhantomIQPassed:             true,
				DetectorFlatfieldCalibrated: false,
			},
			wantValid:     false,
			expectedFails: 1,
		},
		{
			name: "Todas las pruebas faltantes / en blanco",
			qa: models.PostRepairQA{
				ZAlignmentPassed:            false,
				PhantomIQPassed:             false,
				DetectorFlatfieldCalibrated: false,
			},
			wantValid:     false,
			expectedFails: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateTicketClosure(tt.qa)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateTicketClosure() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if len(result.MissingChecks) != tt.expectedFails {
				t.Errorf("ValidateTicketClosure() MissingChecks count = %d, want %d", len(result.MissingChecks), tt.expectedFails)
			}
			if !result.Valid && result.ErrorMessage == "" {
				t.Errorf("ValidateTicketClosure() esperaba ErrorMessage no vacío ante validación fallida")
			}
		})
	}
}

func TestIsCalibrationFullyPassed(t *testing.T) {
	if !IsCalibrationFullyPassed(models.PostRepairQA{
		ZAlignmentPassed:            true,
		PhantomIQPassed:             true,
		DetectorFlatfieldCalibrated: true,
	}) {
		t.Errorf("IsCalibrationFullyPassed debió ser true cuando las 3 pruebas son verdaderas")
	}

	if IsCalibrationFullyPassed(models.PostRepairQA{
		ZAlignmentPassed:            true,
		PhantomIQPassed:             false,
		DetectorFlatfieldCalibrated: true,
	}) {
		t.Errorf("IsCalibrationFullyPassed debió ser false si falta PhantomIQ")
	}
}
