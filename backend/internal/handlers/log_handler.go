package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"mitfv2/internal/database"
	"mitfv2/internal/parser"
	"mitfv2/internal/sshclient"
)

// Constante con el nombre del archivo de logs por defecto
const GesysCT99LogFile = "gesys_ct99.log"

// LogHandler maneja las peticiones HTTP relacionadas con logs y SSH.
type LogHandler struct {
	client *sshclient.SSHClient
	repo   *database.DBRepository
}

// NewLogHandler instancia el handler con el cliente SSH y repositorio de configuraciones opcional.
func NewLogHandler(client *sshclient.SSHClient, repo *database.DBRepository) *LogHandler {
	return &LogHandler{
		client: client,
		repo:   repo,
	}
}

// getClientForRequest determina el cliente SSH a usar según el parámetro ?server_id o la configuración global/activa.
func (h *LogHandler) getClientForRequest(r *http.Request) (*sshclient.SSHClient, *database.ServerConfig, error) {
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

// JSONResponse estructura estándar auxiliar para respuestas simples.
type JSONResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func sendJSON(w http.ResponseWriter, status int, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// GetLogsHandler procesa la búsqueda, filtrado y paginación de logs del sistema.
// Endpoint: GET /api/logs?severity=3&host=tgp&search=STOP%20SCAN&page=1&limit=20&server_id=1
// Parámetros soportados:
//   - severity: Filtra por nivel (ej: "2", "3", "Pri/Soft", "Error", "Warn")
//   - host: Filtra por el host o subsistema (ej: "tubemgr", "tgp")
//   - search: Búsqueda por coincidencia en mensaje, código Ermes, archivo .cxx, etc.
//   - device: Filtra por componente de hardware (ej: "X-Ray Tube")
//   - date: Filtra por fecha (ej: "2026-09-03", "Sep 3")
//   - page: Número de página (Default: 1)
//   - limit: Cantidad por página (Default: 20, Max: 100)
//   - file: Archivo a leer (opcional, default: "gesys_ct99.log")
//   - lines: Cantidad de líneas a leer desde el final (opcional, default: 0 para todo el archivo)
//   - server_id: ID opcional del servidor en la base de datos (ej: ?server_id=2)
func (h *LogHandler) GetLogsHandler(w http.ResponseWriter, r *http.Request) {
	filterOpts := parser.ParseFilterOptions(r)

	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	targetFile := r.URL.Query().Get("file")
	if targetFile == "" {
		targetFile = GesysCT99LogFile
	}

	lines := 0 // 0 = leer el archivo completo
	if lStr := r.URL.Query().Get("lines"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l >= 0 {
			lines = l
		}
	}

	content, err := client.ReadLogFile(targetFile, lines)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Error al leer archivo de log %s: %v", targetFile, err),
			"data":  []parser.GELogRecord{},
			"pagination": parser.PaginationMeta{
				TotalRecords: 0,
				CurrentPage:  filterOpts.Page,
				Limit:        filterOpts.Limit,
				TotalPages:   0,
				HasNext:      false,
				HasPrev:      false,
			},
		})
		return
	}

	records, err := parser.ParseCT99String(content, 0)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Error parseando contenido de log: %v", err),
			"data":  []parser.GELogRecord{},
			"pagination": parser.PaginationMeta{
				TotalRecords: 0,
				CurrentPage:  filterOpts.Page,
				Limit:        filterOpts.Limit,
				TotalPages:   0,
				HasNext:      false,
				HasPrev:      false,
			},
		})
		return
	}

	// 1. Aplicar filtros en memoria
	filtered := parser.ApplyFilters(records, filterOpts)

	// 2. Aplicar paginación segura
	paged, paginationMeta := parser.PaginateRecords(filtered, filterOpts.Page, filterOpts.Limit)
	if paged == nil {
		paged = []parser.GELogRecord{}
	}

	target := client.GetTarget()
	serverInfo := map[string]interface{}{
		"host": target.SSHHost,
		"port": target.SSHPort,
		"user": target.SSHUser,
	}
	if serverCfg != nil {
		serverInfo["id"] = serverCfg.ID
		serverInfo["name"] = serverCfg.Name
	}

	// 3. Respuesta JSON esperada por Next.js
	response := map[string]interface{}{
		"data":        paged,
		"pagination":  paginationMeta,
		"server_info": serverInfo,
	}

	sendJSON(w, http.StatusOK, response)
}

// HealthHandler devuelve el estado del backend y la configuración actual.
// Soporta ?server_id=X para verificar el estado de un servidor en específico.
func (h *LogHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	t := client.GetTarget()
	data := map[string]interface{}{
		"ssh_host":        t.SSHHost,
		"ssh_port":        t.SSHPort,
		"ssh_user":        t.SSHUser,
		"remote_log_path": t.RemoteLogPath,
		"auth_method": func() string {
			if t.SSHKeyPath != "" {
				return "private_key"
			}
			if t.SSHPassword != "" {
				return "password"
			}
			return "none"
		}(),
	}
	if serverCfg != nil {
		data["server_id"] = serverCfg.ID
		data["server_name"] = serverCfg.Name
		data["is_active"] = serverCfg.IsActive
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: "Backend Go MITFV2 activo",
		Data:    data,
	})
}

// TestSSHHandler verifica la conexión SSH contra el servidor Linux.
// Soporta ?server_id=X para verificar cualquier servidor registrado.
func (h *LogHandler) TestSSHHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	t := client.GetTarget()
	output, err := client.TestConnection()
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("Fallo de conexión SSH con %s@%s:%d: %v", t.SSHUser, t.SSHHost, t.SSHPort, err),
		})
		return
	}

	dataResp := map[string]interface{}{
		"server_info": output,
		"ssh_host":    t.SSHHost,
		"ssh_port":    t.SSHPort,
		"ssh_user":    t.SSHUser,
	}
	if serverCfg != nil {
		dataResp["server_id"] = serverCfg.ID
		dataResp["server_name"] = serverCfg.Name
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("Conexión SSH exitosa con %s (%s@%s:%d)", func() string {
			if serverCfg != nil {
				return serverCfg.Name
			}
			return t.SSHHost
		}(), t.SSHUser, t.SSHHost, t.SSHPort),
		Data: dataResp,
	})
}

// LsHandler ejecuta un 'ls' en la ruta remota configurada o en un subpath.
// Soporta ?server_id=X para explorar el filesystem de un servidor específico.
func (h *LogHandler) LsHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	subPath := r.URL.Query().Get("path")
	if subPath == "" {
		subPath = r.URL.Query().Get("subpath")
	}

	listing, err := client.ListDirectory(subPath)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("Error ejecutando ls en el servidor remoto: %v", err),
		})
		return
	}

	resData := map[string]interface{}{
		"listing": listing,
	}
	if serverCfg != nil {
		resData["server_id"] = serverCfg.ID
		resData["server_name"] = serverCfg.Name
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("Listado de archivos en %s", listing.RemotePath),
		Data:    resData,
	})
}

// GesysCT99LogHandler lee el archivo gesys_ct99.log en RemoteLogPath.
// Si se incluye ?parsed=true redirige a GetLogsHandler.
// Soporta ?server_id=X.
func (h *LogHandler) GesysCT99LogHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("parsed") == "true" {
		h.GetLogsHandler(w, r)
		return
	}

	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	lines := 100
	if lStr := r.URL.Query().Get("lines"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l >= 0 {
			lines = l
		}
	}

	content, err := client.ReadLogFile(GesysCT99LogFile, lines)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("Error al leer %s: %v", GesysCT99LogFile, err),
		})
		return
	}

	if r.URL.Query().Get("raw") == "true" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
		return
	}

	t := client.GetTarget()
	dataResp := map[string]interface{}{
		"file":        GesysCT99LogFile,
		"remote_path": t.RemoteLogPath + "/" + GesysCT99LogFile,
		"lines":       lines,
		"content":     content,
		"ssh_host":    t.SSHHost,
	}
	if serverCfg != nil {
		dataResp["server_id"] = serverCfg.ID
		dataResp["server_name"] = serverCfg.Name
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("Archivo %s leído exitosamente", GesysCT99LogFile),
		Data:    dataResp,
	})
}

// ParseRawTextHandler endpoint POST que recibe texto crudo de log y lo retorna filtrado/parseado a JSON.
func (h *LogHandler) ParseRawTextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, JSONResponse{
			Success: false,
			Error:   "Método no permitido. Utiliza POST.",
		})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   "Error al leer cuerpo de la petición",
		})
		return
	}
	defer r.Body.Close()

	records, err := parser.ParseCT99String(string(bodyBytes), 0)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("Error parseando texto: %v", err),
		})
		return
	}

	filterOpts := parser.ParseFilterOptions(r)
	filtered := parser.ApplyFilters(records, filterOpts)
	paged, paginationMeta := parser.PaginateRecords(filtered, filterOpts.Page, filterOpts.Limit)
	if paged == nil {
		paged = []parser.GELogRecord{}
	}

	sendJSON(w, http.StatusOK, parser.PaginatedLogsResponse{
		Data:       paged,
		Pagination: paginationMeta,
	})
}

// ListLogsHandler lista los archivos de log disponibles en la ruta remota.
// Soporta ?server_id=X.
func (h *LogHandler) ListLogsHandler(w http.ResponseWriter, r *http.Request) {
	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	files, err := client.ListLogFiles()
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("Error al listar archivos de logs: %v", err),
		})
		return
	}

	t := client.GetTarget()
	dataResp := map[string]interface{}{
		"remote_path": t.RemoteLogPath,
		"total":       len(files),
		"files":       files,
		"ssh_host":    t.SSHHost,
	}
	if serverCfg != nil {
		dataResp["server_id"] = serverCfg.ID
		dataResp["server_name"] = serverCfg.Name
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: "Archivos de logs obtenidos",
		Data:    dataResp,
	})
}

// ReadLogHandler lee un archivo arbitrario mediante query param ?file=xxx.
// Soporta ?server_id=X.
func (h *LogHandler) ReadLogHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   "Parámetro requerido faltante: ?file=<nombre_del_log>",
		})
		return
	}

	client, serverCfg, err := h.getClientForRequest(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	lines := 100
	if lStr := r.URL.Query().Get("lines"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l >= 0 {
			lines = l
		}
	}

	content, err := client.ReadLogFile(fileName, lines)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("Error leyendo archivo %s: %v", fileName, err),
		})
		return
	}

	if r.URL.Query().Get("raw") == "true" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
		return
	}

	t := client.GetTarget()
	dataResp := map[string]interface{}{
		"file":        fileName,
		"remote_path": t.RemoteLogPath + "/" + fileName,
		"lines":       lines,
		"content":     content,
		"ssh_host":    t.SSHHost,
	}
	if serverCfg != nil {
		dataResp["server_id"] = serverCfg.ID
		dataResp["server_name"] = serverCfg.Name
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("Archivo %s leído exitosamente", fileName),
		Data:    dataResp,
	})
}

// StreamLogHandler realiza streaming en vivo de un archivo de log vía SSE (Server-Sent Events).
// Soporta ?server_id=X.
func (h *LogHandler) StreamLogHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   "Parámetro requerido faltante: ?file=<nombre_del_log>",
		})
		return
	}

	client, _, err := h.getClientForRequest(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	lines := 50
	if lStr := r.URL.Query().Get("lines"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			lines = l
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		sendJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   "Streaming no soportado por este cliente HTTP",
		})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lineChan := make(chan string, 100)
	errChan := make(chan error, 1)

	ctx := r.Context()
	go func() {
		errChan <- client.StreamLogFile(ctx, fileName, lines, lineChan)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-lineChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		case err := <-errChan:
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
				flusher.Flush()
			}
			return
		}
	}
}
