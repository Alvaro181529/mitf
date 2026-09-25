package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mitfv2/internal/database"
	"mitfv2/internal/parser"
	"mitfv2/internal/sshclient"
	"mitfv2/services"
)

// sendError responde con un formato estándar de error JSON.
func sendError(w http.ResponseWriter, status int, msg string) {
	sendJSON(w, status, map[string]interface{}{
		"status":  "error",
		"message": msg,
	})
}

// HardwareHandler maneja las peticiones de métricas de hardware y alertas TSM.
type HardwareHandler struct {
	client *sshclient.SSHClient
	repo   *database.DBRepository
}

// NewHardwareHandler crea una instancia del handler de hardware con soporte multi-servidor opcional.
func NewHardwareHandler(client *sshclient.SSHClient, repo *database.DBRepository) *HardwareHandler {
	return &HardwareHandler{
		client: client,
		repo:   repo,
	}
}

// getClientForRequest determina el cliente SSH y metadatos a usar según el parámetro ?server_id.
func (h *HardwareHandler) getClientForRequest(r *http.Request) (*sshclient.SSHClient, *database.ServerConfig, error) {
	if sIDStr := r.URL.Query().Get("server_id"); sIDStr != "" && h.repo != nil {
		sID, err := strconv.ParseInt(sIDStr, 10, 64)
		if err != nil || sID <= 0 {
			return nil, nil, fmt.Errorf("parámetro 'server_id' inválido: %s", sIDStr)
		}
		cfg, err := h.repo.GetByID(sID)
		if err != nil {
			return nil, nil, fmt.Errorf("servidor con ID %d no encontrado: %w", sID, err)
		}
		target := sshclient.ServerTarget{
			SSHHost:          cfg.SSHHost,
			SSHPort:          cfg.SSHPort,
			SSHUser:          cfg.SSHUser,
			SSHPassword:      cfg.SSHPassword,
			SSHKeyPath:       cfg.SSHKeyPath,
			SSHKeyPassphrase: cfg.SSHKeyPassphrase,
			RemoteLogPath:    cfg.RemoteLogPath,
		}
		return sshclient.NewSSHClientForTarget(target), cfg, nil
	}

	return h.client, nil, nil
}

// loadRecords obtiene y procesa los registros de gesys_ct99.log usando el cliente SSH especificado.
func (h *HardwareHandler) loadRecords(client *sshclient.SSHClient, lines int) ([]services.LogEntry, error) {
	content, err := client.ReadLogFile(GesysCT99LogFile, lines)
	if err != nil {
		return nil, err
	}
	return parser.ParseCT99String(content, 0)
}

// fetchTelemetry ejecuta una extracción directa y rápida de la telemetría en gesys_ct99.log
func (h *HardwareHandler) fetchTelemetry(client *sshclient.SSHClient) (services.TubeHealthData, services.GantryStatsData, []services.TSMSubsystemHealth) {
	target := client.GetTarget()

	// 1. Cargar registros recientes para el análisis de subsistemas TSM
	records, _ := h.loadRecords(client, 1000)

	// 2. Extraer métricas base desde los registros parseados
	tube, gantry, tsmHealth := services.ExtractHardwareMetrics(records)

	// 3. Consultar telemetría directa desde el log del servidor remoto con un comando optimizado
	cmd := fmt.Sprintf(`tac "%s/gesys_ct99.log" | awk '
/reports a total of/ && !mas { mas=$0 }
/Total No. of Gantry Revolutions/ && !revs { revs=$0 }
/Tube temperature/ && !temp { temp=$0 }
mas && revs && temp { print mas; print revs; print temp; exit }
END { if (mas) print mas; if (revs) print revs; if (temp) print temp }
'`, target.RemoteLogPath)

	if out, err := client.ExecuteCommand(cmd); err == nil && strings.TrimSpace(out) != "" {
		mas, revs, temp, mhu := services.ParseRawTelemetryOutput(out)
		if mas > 0 {
			tube.AccumulatedMas = mas
			tube.UsagePercentage = services.CalculateTubeEOL(mas)
		}
		if temp > 0 {
			tube.AnodeTemperatureCelsius = temp
			if mhu > 0 {
				tube.AnodeThermalCapacityMHU = mhu
			}
			tube.Status = services.EvaluateAnodeThermalRisk(temp, tube.AnodeThermalCapacityMHU)
		}
		if revs > 0 {
			gantry.AccumulatedRevolutions = revs
			gantry.UsagePercentage = services.CalculateRotorLifecycle(revs)
			if gantry.UsagePercentage >= 100.0 {
				gantry.Status = "OPERATIONAL"
			}
		}
	}

	return tube, gantry, tsmHealth
}

// buildServerInfo crea el mapa de metadatos del servidor consultado.
func buildServerInfo(client *sshclient.SSHClient, cfg *database.ServerConfig) map[string]interface{} {
	t := client.GetTarget()
	info := map[string]interface{}{
		"host": t.SSHHost,
		"port": t.SSHPort,
		"user": t.SSHUser,
	}
	if cfg != nil {
		info["id"] = cfg.ID
		info["name"] = cfg.Name
		info["is_active"] = cfg.IsActive
	}
	return info
}

// TubeHealthHandler devuelve las métricas y porcentaje consumido del tubo de Rayos X.
// GET /api/v1/hardware/tube-health?server_id=1
func (h *HardwareHandler) TubeHealthHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	tube, _, _ := h.fetchTelemetry(client)

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"server_info": buildServerInfo(client, serverCfg),
		"data":        tube,
	})
}

// GantryStatsHandler devuelve las estadísticas del rotor y cinemática del gantry.
// GET /api/v1/hardware/gantry-stats?server_id=1
func (h *HardwareHandler) GantryStatsHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, gantry, _ := h.fetchTelemetry(client)

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"server_info": buildServerInfo(client, serverCfg),
		"data":        gantry,
	})
}

// HardwareSummaryHandler devuelve el resumen global de salud para el Dashboard Principal.
// GET /api/v1/hardware/summary?server_id=1
func (h *HardwareHandler) HardwareSummaryHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	tube, gantry, tsmHealth := h.fetchTelemetry(client)

	response := map[string]interface{}{
		"status":      "success",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"server_info": buildServerInfo(client, serverCfg),
		"data": services.HardwareSummaryData{
			TubeRX:              tube,
			GantryRotor:         gantry,
			TSMSubsystemsHealth: tsmHealth,
		},
	}

	sendJSON(w, http.StatusOK, response)
}

// InteractiveMapHandler expone el estado de salud y métricas operativas de cada nodo/subcomponente físico.
// GET /api/v1/hardware/interactive-map?server_id=1
func (h *HardwareHandler) InteractiveMapHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	tube, gantry, _ := h.fetchTelemetry(client)
	records, _ := h.loadRecords(client, 1000)

	systemStatus, components := services.BuildInteractiveMap(records, tube, gantry)

	response := map[string]interface{}{
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"system_status": systemStatus,
		"server_info":   buildServerInfo(client, serverCfg),
		"components":    components,
	}

	sendJSON(w, http.StatusOK, response)
}

// fetchWarmupData extrae y analiza las rutinas de calentamiento de tubo en gesys_ct99.log.
func (h *HardwareHandler) fetchWarmupData(client *sshclient.SSHClient) ([]services.WarmupRoutineCycle, []services.SkippedWarmupEvent, services.WarmupSummary, error) {
	target := client.GetTarget()

	// 1. Ejecutar comando python3 optimizado en el servidor remoto Linux para extraer los bloques SR/EN de calentamiento
	pyCmd := fmt.Sprintf(`python3 -c '
import re
path = "%s/gesys_ct99.log"
with open(path, "r", encoding="utf-8", errors="ignore") as f:
    text = f.read()
blocks = re.findall(r"(SR\s+\d+.*?EN\s+\d+)", text, re.DOTALL)
warm_blocks = [b for b in blocks if "Cold Tube Warmup" in b or "WarmupII" in b or "Tube warm up was skipped" in b]
print("\n".join(warm_blocks))
'
`, target.RemoteLogPath)

	raw, err := client.ExecuteCommand(pyCmd)
	if err == nil && strings.TrimSpace(raw) != "" {
		records, parseErr := parser.ParseCT99String(raw, 0)
		if parseErr == nil && len(records) > 0 {
			routines, skipped, summary := services.ParseWarmupRecords(records)
			return routines, skipped, summary, nil
		}
	}

	// 2. Fallback: Si python3 falla, leer el log mediante ReadLogFile y parsearlo directamente
	content, err := client.ReadLogFile(GesysCT99LogFile, 0)
	if err != nil {
		return nil, nil, services.WarmupSummary{}, err
	}
	records, err := parser.ParseCT99String(content, 0)
	if err != nil {
		return nil, nil, services.WarmupSummary{}, err
	}
	routines, skipped, summary := services.ParseWarmupRecords(records)
	return routines, skipped, summary, nil
}

// TubeWarmupHandler analiza gesys_ct99.log y retorna las Rutinas de Calentamiento de Tubo (Warm-up Routines).
// GET /api/v1/hardware/tube-warmup?server_id=1&limit=50
func (h *HardwareHandler) TubeWarmupHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	routines, skipped, summary, err := h.fetchWarmupData(client)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error procesando rutinas de calentamiento: %v", err))
		return
	}

	// Opcional: limitar cantidad de rutinas devueltas si el usuario especifica ?limit=N
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if limit, err := strconv.Atoi(lStr); err == nil && limit > 0 && limit < len(routines) {
			routines = routines[:limit]
		}
	}

	resp := services.TubeWarmupResponse{
		Status:        "success",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		ServerInfo:    buildServerInfo(client, serverCfg),
		Summary:       summary,
		Routines:      routines,
		SkippedEvents: skipped,
	}

	sendJSON(w, http.StatusOK, resp)
}

// TSMAlertsEventsHandler devuelve las alertas filtradas de los 5 subsistemas TSM.
// GET /api/v1/alerts/events?severity=2&tsm_subsystem=1&limit=50&server_id=1
// Nota: Por defecto ignora severidad 1 (ruido de auditoría).
func (h *HardwareHandler) TSMAlertsEventsHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	records, err := h.loadRecords(client, 0)
	if err != nil {
		records = []services.LogEntry{}
	}

	// 1. Filtrar ruido: ignorar severidad 1 (Info) por defecto a menos que se pida explícitamente
	reqSeverity := strings.TrimSpace(r.URL.Query().Get("severity"))
	var filtered []services.LogEntry

	if reqSeverity != "" {
		filtered = parser.ApplyFilters(records, parser.FilterOptions{Severity: reqSeverity})
	} else {
		for _, rec := range records {
			if rec.SeverityCode >= 2 {
				filtered = append(filtered, rec)
			}
		}
	}

	// 2. Filtrar por subsistema TSM (1 a 5)
	if subStr := r.URL.Query().Get("tsm_subsystem"); subStr != "" {
		if subID, err := strconv.Atoi(subStr); err == nil && subID >= 1 && subID <= 5 {
			filtered = services.FilterTSMAlerts(filtered, subID)
		}
	}

	// 3. Paginación
	filterOpts := parser.ParseFilterOptions(r)
	paged, meta := parser.PaginateRecords(filtered, filterOpts.Page, filterOpts.Limit)
	if paged == nil {
		paged = []services.LogEntry{}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"server_info": buildServerInfo(client, serverCfg),
		"data":        paged,
		"pagination":  meta,
	})
}

// TSMAlertsBySubsystemHandler devuelve las alertas filtradas para un subsistema específico en la URL.
// GET /api/v1/alerts/tsm/{subsystem_id}?server_id=1
func (h *HardwareHandler) TSMAlertsBySubsystemHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/tsm/")
	subsystemID, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil || subsystemID < 1 || subsystemID > 5 {
		sendError(w, http.StatusBadRequest, "El parámetro subsystem_id debe ser un entero entre 1 y 5")
		return
	}

	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	records, err := h.loadRecords(client, 0)
	if err != nil {
		records = []services.LogEntry{}
	}

	// Filtrar por subsistema solicitado
	matched := services.FilterTSMAlerts(records, subsystemID)

	// Filtrar ruido de severidad 1 si no se solicitó
	reqSeverity := strings.TrimSpace(r.URL.Query().Get("severity"))
	var filtered []services.LogEntry
	if reqSeverity != "" {
		filtered = parser.ApplyFilters(matched, parser.FilterOptions{Severity: reqSeverity})
	} else {
		for _, rec := range matched {
			if rec.SeverityCode >= 2 {
				filtered = append(filtered, rec)
			}
		}
	}

	// Paginación
	filterOpts := parser.ParseFilterOptions(r)
	paged, meta := parser.PaginateRecords(filtered, filterOpts.Page, filterOpts.Limit)
	if paged == nil {
		paged = []services.LogEntry{}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "success",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"server_info":    buildServerInfo(client, serverCfg),
		"subsystem_id":   subsystemID,
		"subsystem_name": services.GetTSMSubsystemName(subsystemID),
		"data":           paged,
		"pagination":     meta,
	})
}

// JediCanLogsHandler maneja la lectura del log CAN bus jedi_can_at_error.log
// GET /api/v1/logs/jedi-can?lines=100&server_id=1
func (h *HardwareHandler) JediCanLogsHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	lines := 100
	if l := r.URL.Query().Get("lines"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			lines = parsed
		}
	}

	content, err := client.ReadLogFile("jedi_can_at_error.log", lines)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error al leer jedi_can_at_error.log: %v", err))
		return
	}

	frames := parser.ParseCANContent(content)

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "success",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"server_info":    buildServerInfo(client, serverCfg),
		"file":           "jedi_can_at_error.log",
		"lines":          lines,
		"subsystem_id":   1,
		"subsystem_name": "Generador y Tubo RX (Subsystem 1)",
		"total_frames":   len(frames),
		"frames":         frames,
		"content":        content,
	})
}

// DasErrorsLogsHandler maneja la lectura de errores del subsistema DAS dataacq.stderr.log
// GET /api/v1/logs/das-errors?lines=100&server_id=1
func (h *HardwareHandler) DasErrorsLogsHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	lines := 100
	if l := r.URL.Query().Get("lines"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			lines = parsed
		}
	}

	content, err := client.ReadLogFile("dataacq.stderr.log", lines)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Error al leer dataacq.stderr.log: %v", err))
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"server_info": buildServerInfo(client, serverCfg),
		"file":        "dataacq.stderr.log",
		"lines":       lines,
		"content":     content,
	})
}
