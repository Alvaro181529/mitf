package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"mitfv2/models"
)

// PostgresATRECRepository implementa ATRECRepository con persistencia directa en PostgreSQL.
type PostgresATRECRepository struct {
	db *sql.DB
}

// NewPostgresATRECRepository crea una nueva instancia persistente de ATRECRepository.
func NewPostgresATRECRepository(db *sql.DB) *PostgresATRECRepository {
	repo := &PostgresATRECRepository{db: db}
	repo.seedInitialTickets()
	return repo
}

// seedInitialTickets siembra tickets iniciales si la tabla atrec_tickets está vacía.
func (r *PostgresATRECRepository) seedInitialTickets() {
	var count int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM atrec_tickets").Scan(&count); err != nil || count > 0 {
		return
	}

	now := time.Now().UTC()
	t1 := models.TicketATREC{
		TicketID:           "ATREC-2026-101",
		CreatedAt:          now.Add(-2 * time.Hour),
		UpdatedAt:          now.Add(-2 * time.Hour),
		TSMSubsystemID:     1,
		Severity:           models.SeverityCritical,
		SourceErrorCode:    "260199001",
		Description:        "Arco de filamento en tubo de rayos X detectado en rotación sostenida",
		AssignedTechnician: "Ing. Carlos Mendoza",
		Status:             models.StatusOpen,
		PostRepairQA: models.PostRepairQA{
			ZAlignmentPassed:            false,
			PhantomIQPassed:             false,
			DetectorFlatfieldCalibrated: false,
			QASignOffBy:                 "",
			ResolutionType:              "",
		},
	}
	t2 := models.TicketATREC{
		TicketID:           "ATREC-2026-102",
		CreatedAt:          now.Add(-5 * time.Hour),
		UpdatedAt:          now.Add(-1 * time.Hour),
		TSMSubsystemID:     2,
		Severity:           models.SeverityMajor,
		SourceErrorCode:    "260144012",
		Description:        "Desviación de jitter en anillos deslizantes del rotor",
		AssignedTechnician: "Tec. Laura Gómez",
		Status:             models.StatusPendingCalibration,
		PostRepairQA: models.PostRepairQA{
			ZAlignmentPassed:            true,
			PhantomIQPassed:             false,
			DetectorFlatfieldCalibrated: false,
			QASignOffBy:                 "Tec. Laura Gómez",
			ResolutionType:              models.ResolutionTroubleshootingL1,
		},
	}

	_, _ = r.Create(t1)
	_, _ = r.Create(t2)
}

// GenerateTicketID genera un ID único correlativo consultando la base de datos PostgreSQL.
func (r *PostgresATRECRepository) GenerateTicketID() string {
	year := time.Now().Year()
	prefix := fmt.Sprintf("ATREC-%d-%%", year)

	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM atrec_tickets WHERE ticket_id LIKE $1", prefix).Scan(&count)
	if err != nil {
		count = 100
	}

	return fmt.Sprintf("ATREC-%d-%03d", year, count+101)
}

// GetAll retorna todos los tickets desde PostgreSQL con orden cronológico descendente.
func (r *PostgresATRECRepository) GetAll(statusFilter string) ([]models.TicketATREC, error) {
	var rows *sql.Rows
	var err error

	cleanStatus := strings.ToUpper(strings.TrimSpace(statusFilter))
	if cleanStatus != "" {
		query := `
			SELECT ticket_id, created_at, updated_at, tsm_subsystem_id, severity,
			       source_error_code, description, assigned_technician, status,
			       z_alignment_passed, phantom_iq_passed, detector_flatfield_calibrated,
			       qa_sign_off_by, resolution_type
			FROM atrec_tickets
			WHERE UPPER(status) = $1
			ORDER BY created_at DESC
		`
		rows, err = r.db.Query(query, cleanStatus)
	} else {
		query := `
			SELECT ticket_id, created_at, updated_at, tsm_subsystem_id, severity,
			       source_error_code, description, assigned_technician, status,
			       z_alignment_passed, phantom_iq_passed, detector_flatfield_calibrated,
			       qa_sign_off_by, resolution_type
			FROM atrec_tickets
			ORDER BY created_at DESC
		`
		rows, err = r.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := make([]models.TicketATREC, 0)
	for rows.Next() {
		var t models.TicketATREC
		err := rows.Scan(
			&t.TicketID, &t.CreatedAt, &t.UpdatedAt, &t.TSMSubsystemID, &t.Severity,
			&t.SourceErrorCode, &t.Description, &t.AssignedTechnician, &t.Status,
			&t.PostRepairQA.ZAlignmentPassed, &t.PostRepairQA.PhantomIQPassed,
			&t.PostRepairQA.DetectorFlatfieldCalibrated, &t.PostRepairQA.QASignOffBy,
			&t.PostRepairQA.ResolutionType,
		)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}

	return tickets, nil
}

// GetByID busca un ticket en PostgreSQL por su ID.
func (r *PostgresATRECRepository) GetByID(ticketID string) (*models.TicketATREC, error) {
	var t models.TicketATREC
	query := `
		SELECT ticket_id, created_at, updated_at, tsm_subsystem_id, severity,
			   source_error_code, description, assigned_technician, status,
			   z_alignment_passed, phantom_iq_passed, detector_flatfield_calibrated,
			   qa_sign_off_by, resolution_type
		FROM atrec_tickets
		WHERE LOWER(ticket_id) = LOWER($1)
	`
	err := r.db.QueryRow(query, strings.TrimSpace(ticketID)).Scan(
		&t.TicketID, &t.CreatedAt, &t.UpdatedAt, &t.TSMSubsystemID, &t.Severity,
		&t.SourceErrorCode, &t.Description, &t.AssignedTechnician, &t.Status,
		&t.PostRepairQA.ZAlignmentPassed, &t.PostRepairQA.PhantomIQPassed,
		&t.PostRepairQA.DetectorFlatfieldCalibrated, &t.PostRepairQA.QASignOffBy,
		&t.PostRepairQA.ResolutionType,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticket con ID '%s' no encontrado", ticketID)
		}
		return nil, err
	}

	return &t, nil
}

// Create inserta un nuevo ticket de forma persistente en PostgreSQL.
func (r *PostgresATRECRepository) Create(ticket models.TicketATREC) (*models.TicketATREC, error) {
	if ticket.CreatedAt.IsZero() {
		ticket.CreatedAt = time.Now().UTC()
	}
	if ticket.UpdatedAt.IsZero() {
		ticket.UpdatedAt = ticket.CreatedAt
	}

	query := `
		INSERT INTO atrec_tickets (
			ticket_id, created_at, updated_at, tsm_subsystem_id, severity,
			source_error_code, description, assigned_technician, status,
			z_alignment_passed, phantom_iq_passed, detector_flatfield_calibrated,
			qa_sign_off_by, resolution_type
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.Exec(
		query,
		ticket.TicketID, ticket.CreatedAt, ticket.UpdatedAt, ticket.TSMSubsystemID, ticket.Severity,
		ticket.SourceErrorCode, ticket.Description, ticket.AssignedTechnician, ticket.Status,
		ticket.PostRepairQA.ZAlignmentPassed, ticket.PostRepairQA.PhantomIQPassed,
		ticket.PostRepairQA.DetectorFlatfieldCalibrated, ticket.PostRepairQA.QASignOffBy,
		ticket.PostRepairQA.ResolutionType,
	)
	if err != nil {
		return nil, fmt.Errorf("error al guardar ticket en PostgreSQL: %w", err)
	}

	return &ticket, nil
}

// Update actualiza un ticket existente en PostgreSQL.
func (r *PostgresATRECRepository) Update(ticket models.TicketATREC) (*models.TicketATREC, error) {
	ticket.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE atrec_tickets
		SET updated_at = $1, tsm_subsystem_id = $2, severity = $3,
		    source_error_code = $4, description = $5, assigned_technician = $6, status = $7,
		    z_alignment_passed = $8, phantom_iq_passed = $9, detector_flatfield_calibrated = $10,
		    qa_sign_off_by = $11, resolution_type = $12
		WHERE LOWER(ticket_id) = LOWER($13)
	`
	res, err := r.db.Exec(
		query,
		ticket.UpdatedAt, ticket.TSMSubsystemID, ticket.Severity,
		ticket.SourceErrorCode, ticket.Description, ticket.AssignedTechnician, ticket.Status,
		ticket.PostRepairQA.ZAlignmentPassed, ticket.PostRepairQA.PhantomIQPassed,
		ticket.PostRepairQA.DetectorFlatfieldCalibrated, ticket.PostRepairQA.QASignOffBy,
		ticket.PostRepairQA.ResolutionType, strings.TrimSpace(ticket.TicketID),
	)
	if err != nil {
		return nil, fmt.Errorf("error al actualizar ticket en PostgreSQL: %w", err)
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return nil, fmt.Errorf("ticket con ID '%s' no encontrado para actualizar", ticket.TicketID)
	}

	return &ticket, nil
}
