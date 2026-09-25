package main

import (
	"log"
	"net/http"

	"mitfv2/config"
	"mitfv2/internal/database"
	"mitfv2/internal/handlers"
	"mitfv2/internal/sshclient"
)

func main() {
	config.LoadConfig()

	dbRepo, err := database.InitDB()
	if err != nil {
		log.Printf("[DB WARNING] No se pudo inicializar PostgreSQL: %v (el sistema funcionará con variables locales)", err)
	}

	sshClient := sshclient.NewSSHClient()
	atrecRepo := database.NewMemoryATRECRepository()

	logHandler := handlers.NewLogHandler(sshClient, dbRepo)
	hardwareHandler := handlers.NewHardwareHandler(sshClient, dbRepo)
	atrecHandler := handlers.NewATRECHandler(atrecRepo)
	configHandler := handlers.NewConfigHandler(dbRepo)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/configs", configHandler.ConfigsRouter)
	mux.HandleFunc("/api/v1/configs/", configHandler.ConfigsRouter)
	mux.HandleFunc("/api/configs", configHandler.ConfigsRouter)
	mux.HandleFunc("/api/configs/", configHandler.ConfigsRouter)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"service":"MITFV2 Multi-Server SSH Linux Log Reader & GE CT99 Parser API","version":"2.3.0","multi_server_supported":true,"query_param":"?server_id=<id>","endpoints":{"health":"/api/health?server_id=1","ssh_test":"/api/ssh/test?server_id=1","configs":"GET,POST /api/v1/configs","active_config":"GET /api/v1/configs/active","config_by_id":"GET,PUT,DELETE /api/v1/configs/:id","activate_config":"POST /api/v1/configs/:id/activate","ls":"/api/ls?server_id=1&path=<subpath>","logs_filtered":"GET /api/logs?server_id=1&severity=<level>&host=<host>&search=<text>&device=<dev>&date=<date>&page=<n>&limit=<n>","hardware_summary":"GET /api/v1/hardware/summary?server_id=1","tube_health":"GET /api/v1/hardware/tube-health?server_id=1","gantry_stats":"GET /api/v1/hardware/gantry-stats?server_id=1","interactive_map":"GET /api/v1/hardware/interactive-map?server_id=1","alerts_events":"GET /api/v1/alerts/events?server_id=1&severity=2&tsm_subsystem=1&limit=50","alerts_tsm":"GET /api/v1/alerts/tsm/:subsystem_id?server_id=1","jedi_can":"GET /api/v1/logs/jedi-can?server_id=1","das_errors":"GET /api/v1/logs/das-errors?server_id=1","atrec_tickets":"GET,POST /api/v1/atrec/tickets (?status=OPEN)","atrec_calibration":"PUT /api/v1/atrec/tickets/:id/calibration","atrec_close":"PATCH /api/v1/atrec/tickets/:id/close"}}` + "\n"))
	})

	mux.HandleFunc("/api/health", logHandler.HealthHandler)
	mux.HandleFunc("/api/ssh/test", logHandler.TestSSHHandler)
	mux.HandleFunc("/api/ls", logHandler.LsHandler)
	mux.HandleFunc("/api/logs/ls", logHandler.LsHandler)

	// Endpoints de logs con Filtros y Paginación (Next.js compatible)
	mux.HandleFunc("/api/logs", logHandler.GetLogsHandler)
	mux.HandleFunc("/api/logs/gesys_ct99/parsed", logHandler.GetLogsHandler)
	mux.HandleFunc("/api/logs/gesys_ct99/json", logHandler.GetLogsHandler)
	mux.HandleFunc("/api/logs/gesys_ct99", logHandler.GesysCT99LogHandler)
	mux.HandleFunc("/api/logs/gesys-ct99", logHandler.GesysCT99LogHandler)
	mux.HandleFunc("/api/logs/gesys_ct99.log", logHandler.GesysCT99LogHandler)
	mux.HandleFunc("/api/parser/ct99", logHandler.ParseRawTextHandler)

	// Endpoints para listar archivos y leer archivos individuales
	mux.HandleFunc("/api/logs/files", logHandler.ListLogsHandler)
	mux.HandleFunc("/api/logs/read", logHandler.ReadLogHandler)
	mux.HandleFunc("/api/logs/stream", logHandler.StreamLogHandler)

	// Endpoints V1: Módulo de Métricas de Hardware (KPIs)
	mux.HandleFunc("/api/v1/hardware/tube-health", hardwareHandler.TubeHealthHandler)
	mux.HandleFunc("/api/v1/hardware/gantry-stats", hardwareHandler.GantryStatsHandler)
	mux.HandleFunc("/api/v1/hardware/summary", hardwareHandler.HardwareSummaryHandler)
	mux.HandleFunc("/api/v1/hardware/interactive-map", hardwareHandler.InteractiveMapHandler)
	mux.HandleFunc("/api/hardware/interactive-map", hardwareHandler.InteractiveMapHandler)

	// Endpoints V1: Módulo de Alertas por Subsistema TSM
	mux.HandleFunc("/api/v1/alerts/events", hardwareHandler.TSMAlertsEventsHandler)
	mux.HandleFunc("/api/v1/alerts/tsm/", hardwareHandler.TSMAlertsBySubsystemHandler)

	// Endpoints V1: Módulo de Telemetría Específica
	mux.HandleFunc("/api/v1/logs/jedi-can", hardwareHandler.JediCanLogsHandler)
	mux.HandleFunc("/api/v1/logs/das-errors", hardwareHandler.DasErrorsLogsHandler)

	// Endpoints V1: Módulo ATREC (Gestión de Bitácora de Mantenimiento y Calibración)
	mux.HandleFunc("/api/v1/atrec/tickets", atrecHandler.TicketsRouter)
	mux.HandleFunc("/api/v1/atrec/tickets/", atrecHandler.TicketsRouter)
	mux.HandleFunc("/api/atrec/tickets", atrecHandler.TicketsRouter)
	mux.HandleFunc("/api/atrec/tickets/", atrecHandler.TicketsRouter)

	serverHandler := handlers.CORSMiddleware(mux)

	log.Printf("==========================================================")
	log.Printf(" Backend MITFV2 Multi-Server iniciado en http://localhost%s", config.ServerPort)
	log.Printf(" API Health:           http://localhost%s/api/health?server_id=1", config.ServerPort)
	log.Printf(" API CRUD Configs:     http://localhost%s/api/v1/configs", config.ServerPort)
	log.Printf(" API Active Config:    http://localhost%s/api/v1/configs/active", config.ServerPort)
	log.Printf(" API Hardware Summary: http://localhost%s/api/v1/hardware/summary?server_id=1", config.ServerPort)
	log.Printf(" API Interactive Map:  http://localhost%s/api/v1/hardware/interactive-map?server_id=1", config.ServerPort)
	log.Printf(" API Tube Health:      http://localhost%s/api/v1/hardware/tube-health?server_id=1", config.ServerPort)
	log.Printf(" API Gantry Stats:     http://localhost%s/api/v1/hardware/gantry-stats?server_id=1", config.ServerPort)
	log.Printf(" API TSM Alerts:       http://localhost%s/api/v1/alerts/events?server_id=1&severity=2", config.ServerPort)
	log.Printf(" API ATREC Tickets:    http://localhost%s/api/v1/atrec/tickets", config.ServerPort)
	log.Printf(" API Logs (Filtros):   http://localhost%s/api/logs?server_id=1&severity=3", config.ServerPort)
	log.Printf(" Servidor Linux Base:  %s@%s:%d", config.SSHUser, config.SSHHost, config.SSHPort)
	log.Printf("==========================================================")

	if err := http.ListenAndServe(config.ServerPort, serverHandler); err != nil {
		log.Fatalf("Error crítico al iniciar el servidor HTTP: %v", err)
	}
}
