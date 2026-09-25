package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"mitfv2/config"

	_ "github.com/lib/pq"
)

// ServerConfig representa la configuración de conexión SSH y logs persistida en PostgreSQL.
type ServerConfig struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	SSHHost          string    `json:"ssh_host"`
	SSHPort          int       `json:"ssh_port"`
	SSHUser          string    `json:"ssh_user"`
	SSHPassword      string    `json:"ssh_password"`
	SSHKeyPath       string    `json:"ssh_key_path"`
	SSHKeyPassphrase string    `json:"ssh_key_passphrase"`
	RemoteLogPath    string    `json:"remote_log_path"`
	ServerPort       string    `json:"server_port"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DBRepository gestiona las operaciones CRUD en la base de datos PostgreSQL.
type DBRepository struct {
	db *sql.DB
}

// NewDBRepository crea un nuevo repositorio de base de datos.
func NewDBRepository(db *sql.DB) *DBRepository {
	return &DBRepository{db: db}
}

// GetDB retorna la instancia subyacente de *sql.DB.
func (r *DBRepository) GetDB() *sql.DB {
	return r.db
}

// InitDB inicializa la conexión con PostgreSQL, ejecuta las migraciones y siembra los valores iniciales.
func InitDB() (*DBRepository, error) {
	dsn := config.GetDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error al abrir conexión PostgreSQL: %w", err)
	}

	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(10 * time.Minute)

	if err := db.Ping(); err != nil {
		// Intento alternativo sin contraseña si falló con contraseña por default trust
		altDSN := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", config.DBUser, config.DBHost, config.DBPort, config.DBName, config.DBSSLMode)
		if altDB, altErr := sql.Open("postgres", altDSN); altErr == nil && altDB.Ping() == nil {
			db = altDB
		} else {
			return nil, fmt.Errorf("no se pudo conectar a PostgreSQL (%s): %w", dsn, err)
		}
	}

	repo := NewDBRepository(db)

	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("error ejecutando migración: %w", err)
	}

	if err := repo.seedInitialConfig(); err != nil {
		log.Printf("[DB WARNING] Error al sembrar configuración inicial: %v", err)
	}

	log.Printf("[DB SUCCESS] Conexión y tablas de PostgreSQL verificadas exitosamente")
	return repo, nil
}

// migrate asegura que las tablas server_configs y atrec_tickets existan en PostgreSQL.
func (r *DBRepository) migrate() error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock(7429124)"); err != nil {
		return err
	}

	query := `
	CREATE TABLE IF NOT EXISTS server_configs (
		id BIGSERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL DEFAULT 'Servidor Principal',
		ssh_host VARCHAR(255) NOT NULL,
		ssh_port INTEGER NOT NULL DEFAULT 22,
		ssh_user VARCHAR(100) NOT NULL,
		ssh_password VARCHAR(255) DEFAULT '',
		ssh_key_path VARCHAR(500) DEFAULT '',
		ssh_key_passphrase VARCHAR(255) DEFAULT '',
		remote_log_path VARCHAR(500) NOT NULL DEFAULT '/home/server/log',
		server_port VARCHAR(50) NOT NULL DEFAULT ':8080',
		is_active BOOLEAN NOT NULL DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_server_configs_is_active ON server_configs(is_active);

	CREATE TABLE IF NOT EXISTS atrec_tickets (
		ticket_id VARCHAR(50) PRIMARY KEY,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		tsm_subsystem_id INTEGER NOT NULL,
		severity VARCHAR(50) NOT NULL,
		source_error_code VARCHAR(100) DEFAULT '',
		description TEXT NOT NULL,
		assigned_technician VARCHAR(200) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
		z_alignment_passed BOOLEAN NOT NULL DEFAULT false,
		phantom_iq_passed BOOLEAN NOT NULL DEFAULT false,
		detector_flatfield_calibrated BOOLEAN NOT NULL DEFAULT false,
		qa_sign_off_by VARCHAR(200) DEFAULT '',
		resolution_type VARCHAR(100) DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_atrec_tickets_status ON atrec_tickets(status);
	`
	if _, err := tx.Exec(query); err != nil {
		return err
	}

	return tx.Commit()
}

// seedInitialConfig inserta los valores del archivo .env si la tabla está vacía.
func (r *DBRepository) seedInitialConfig() error {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM server_configs").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		initial := &ServerConfig{
			Name:             "Servidor Principal CT99",
			SSHHost:          config.SSHHost,
			SSHPort:          config.SSHPort,
			SSHUser:          config.SSHUser,
			SSHPassword:      config.SSHPassword,
			SSHKeyPath:       config.SSHKeyPath,
			SSHKeyPassphrase: config.SSHKeyPassphrase,
			RemoteLogPath:    config.RemoteLogPath,
			ServerPort:       config.ServerPort,
			IsActive:         true,
		}
		_, err := r.Create(initial)
		if err != nil {
			return fmt.Errorf("error sembrando valores iniciales: %w", err)
		}
		log.Println("[DB SEED] Configuración inicial de .env guardada exitosamente en PostgreSQL")
	} else {
		// Sincronizar en memoria la configuración activa si ya existía
		active, err := r.GetActive()
		if err == nil && active != nil {
			config.UpdateSSHConfig(active.SSHHost, active.SSHPort, active.SSHUser, active.SSHPassword, active.SSHKeyPath, active.SSHKeyPassphrase, active.RemoteLogPath)
		}
	}

	return nil
}

// GetAll retorna todas las configuraciones guardadas en PostgreSQL.
func (r *DBRepository) GetAll() ([]ServerConfig, error) {
	rows, err := r.db.Query(`
		SELECT id, name, ssh_host, ssh_port, ssh_user, ssh_password, ssh_key_path, ssh_key_passphrase, remote_log_path, server_port, is_active, created_at, updated_at
		FROM server_configs
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []ServerConfig
	for rows.Next() {
		var c ServerConfig
		if err := rows.Scan(
			&c.ID, &c.Name, &c.SSHHost, &c.SSHPort, &c.SSHUser, &c.SSHPassword,
			&c.SSHKeyPath, &c.SSHKeyPassphrase, &c.RemoteLogPath, &c.ServerPort,
			&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}

	if configs == nil {
		configs = []ServerConfig{}
	}
	return configs, nil
}

// GetByID busca una configuración por su identificador único.
func (r *DBRepository) GetByID(id int64) (*ServerConfig, error) {
	var c ServerConfig
	query := `
		SELECT id, name, ssh_host, ssh_port, ssh_user, ssh_password, ssh_key_path, ssh_key_passphrase, remote_log_path, server_port, is_active, created_at, updated_at
		FROM server_configs
		WHERE id = $1
	`
	err := r.db.QueryRow(query, id).Scan(
		&c.ID, &c.Name, &c.SSHHost, &c.SSHPort, &c.SSHUser, &c.SSHPassword,
		&c.SSHKeyPath, &c.SSHKeyPassphrase, &c.RemoteLogPath, &c.ServerPort,
		&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuración con ID %d no encontrada", id)
		}
		return nil, err
	}
	return &c, nil
}

// GetActive retorna la configuración que se encuentra actualmente activa.
func (r *DBRepository) GetActive() (*ServerConfig, error) {
	var c ServerConfig
	query := `
		SELECT id, name, ssh_host, ssh_port, ssh_user, ssh_password, ssh_key_path, ssh_key_passphrase, remote_log_path, server_port, is_active, created_at, updated_at
		FROM server_configs
		WHERE is_active = true
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`
	err := r.db.QueryRow(query).Scan(
		&c.ID, &c.Name, &c.SSHHost, &c.SSHPort, &c.SSHUser, &c.SSHPassword,
		&c.SSHKeyPath, &c.SSHKeyPassphrase, &c.RemoteLogPath, &c.ServerPort,
		&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// Create inserta un nuevo registro de configuración en PostgreSQL.
func (r *DBRepository) Create(cfg *ServerConfig) (*ServerConfig, error) {
	if strings.TrimSpace(cfg.SSHHost) == "" {
		return nil, fmt.Errorf("el campo 'ssh_host' es requerido")
	}
	if strings.TrimSpace(cfg.SSHUser) == "" {
		return nil, fmt.Errorf("el campo 'ssh_user' es requerido")
	}
	if cfg.SSHPort <= 0 {
		cfg.SSHPort = 22
	}
	if strings.TrimSpace(cfg.RemoteLogPath) == "" {
		cfg.RemoteLogPath = "/home/server/log"
	}
	if strings.TrimSpace(cfg.ServerPort) == "" {
		cfg.ServerPort = ":8080"
	}
	if strings.TrimSpace(cfg.Name) == "" {
		cfg.Name = fmt.Sprintf("Servidor %s", cfg.SSHHost)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if cfg.IsActive {
		if _, err := tx.Exec("UPDATE server_configs SET is_active = false"); err != nil {
			return nil, err
		}
	}

	query := `
		INSERT INTO server_configs (name, ssh_host, ssh_port, ssh_user, ssh_password, ssh_key_path, ssh_key_passphrase, remote_log_path, server_port, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(
		query,
		cfg.Name, cfg.SSHHost, cfg.SSHPort, cfg.SSHUser, cfg.SSHPassword,
		cfg.SSHKeyPath, cfg.SSHKeyPassphrase, cfg.RemoteLogPath, cfg.ServerPort, cfg.IsActive,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if cfg.IsActive {
		config.UpdateSSHConfig(cfg.SSHHost, cfg.SSHPort, cfg.SSHUser, cfg.SSHPassword, cfg.SSHKeyPath, cfg.SSHKeyPassphrase, cfg.RemoteLogPath)
	}

	return cfg, nil
}

// Update modifica una configuración existente por su ID.
func (r *DBRepository) Update(id int64, cfg *ServerConfig) (*ServerConfig, error) {
	existing, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.Name) != "" {
		existing.Name = cfg.Name
	}
	if strings.TrimSpace(cfg.SSHHost) != "" {
		existing.SSHHost = cfg.SSHHost
	}
	if cfg.SSHPort > 0 {
		existing.SSHPort = cfg.SSHPort
	}
	if strings.TrimSpace(cfg.SSHUser) != "" {
		existing.SSHUser = cfg.SSHUser
	}
	// Password y llaves se actualizan si se especifican
	existing.SSHPassword = cfg.SSHPassword
	existing.SSHKeyPath = cfg.SSHKeyPath
	existing.SSHKeyPassphrase = cfg.SSHKeyPassphrase

	if strings.TrimSpace(cfg.RemoteLogPath) != "" {
		existing.RemoteLogPath = cfg.RemoteLogPath
	}
	if strings.TrimSpace(cfg.ServerPort) != "" {
		existing.ServerPort = cfg.ServerPort
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if cfg.IsActive {
		if _, err := tx.Exec("UPDATE server_configs SET is_active = false"); err != nil {
			return nil, err
		}
		existing.IsActive = true
	}

	query := `
		UPDATE server_configs
		SET name = $1, ssh_host = $2, ssh_port = $3, ssh_user = $4, ssh_password = $5,
		    ssh_key_path = $6, ssh_key_passphrase = $7, remote_log_path = $8,
		    server_port = $9, is_active = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $11
		RETURNING updated_at
	`
	err = tx.QueryRow(
		query,
		existing.Name, existing.SSHHost, existing.SSHPort, existing.SSHUser,
		existing.SSHPassword, existing.SSHKeyPath, existing.SSHKeyPassphrase,
		existing.RemoteLogPath, existing.ServerPort, existing.IsActive, id,
	).Scan(&existing.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if existing.IsActive {
		config.UpdateSSHConfig(existing.SSHHost, existing.SSHPort, existing.SSHUser, existing.SSHPassword, existing.SSHKeyPath, existing.SSHKeyPassphrase, existing.RemoteLogPath)
	}

	return existing, nil
}

// Delete elimina una configuración por ID.
func (r *DBRepository) Delete(id int64) error {
	existing, err := r.GetByID(id)
	if err != nil {
		return err
	}

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM server_configs").Scan(&total); err == nil && total <= 1 {
		return fmt.Errorf("no es posible eliminar la única configuración disponible en la base de datos")
	}

	query := "DELETE FROM server_configs WHERE id = $1"
	_, err = r.db.Exec(query, id)
	if err != nil {
		return err
	}

	// Si se eliminó la activa, marcar la primera disponible como activa
	if existing.IsActive {
		var nextID int64
		if err := r.db.QueryRow("SELECT id FROM server_configs ORDER BY id ASC LIMIT 1").Scan(&nextID); err == nil {
			r.SetActive(nextID)
		}
	}

	return nil
}

// SetActive establece una configuración como activa y actualiza las credenciales SSH en caliente.
func (r *DBRepository) SetActive(id int64) (*ServerConfig, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE server_configs SET is_active = false"); err != nil {
		return nil, err
	}

	res, err := tx.Exec("UPDATE server_configs SET is_active = true, updated_at = CURRENT_TIMESTAMP WHERE id = $1", id)
	if err != nil {
		return nil, err
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return nil, fmt.Errorf("configuración con ID %d no existe", id)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	active, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	config.UpdateSSHConfig(active.SSHHost, active.SSHPort, active.SSHUser, active.SSHPassword, active.SSHKeyPath, active.SSHKeyPassphrase, active.RemoteLogPath)

	return active, nil
}
