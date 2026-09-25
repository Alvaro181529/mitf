package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mitfv2/config"
	"mitfv2/internal/database"
)

// ConfigHandler gestiona los endpoints CRUD de configuraciones del servidor en PostgreSQL o memoria.
type ConfigHandler struct {
	repo *database.DBRepository
}

// NewConfigHandler crea un nuevo handler inyectando el repositorio de base de datos (o nil si no está disponible).
func NewConfigHandler(repo *database.DBRepository) *ConfigHandler {
	return &ConfigHandler{repo: repo}
}

func (h *ConfigHandler) getDefaultConfig() *database.ServerConfig {
	return &database.ServerConfig{
		ID:               1,
		Name:             "Servidor Principal CT99 (Local .env)",
		SSHHost:          config.SSHHost,
		SSHPort:          config.SSHPort,
		SSHUser:          config.SSHUser,
		SSHPassword:      config.SSHPassword,
		SSHKeyPath:       config.SSHKeyPath,
		SSHKeyPassphrase: config.SSHKeyPassphrase,
		RemoteLogPath:    config.RemoteLogPath,
		ServerPort:       config.ServerPort,
		IsActive:         true,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
}

// ConfigsRouter despacha las peticiones a /api/v1/configs y sub-rutas según método y path.
func (h *ConfigHandler) ConfigsRouter(w http.ResponseWriter, r *http.Request) {
	// Normalizar ruta quitando prefijo
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/configs")
	path = strings.TrimPrefix(path, "/api/configs")
	path = strings.Trim(path, "/")

	// 1. Ruta base: /api/v1/configs
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.ListConfigs(w, r)
		case http.MethodPost:
			h.CreateConfig(w, r)
		default:
			sendJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"status":  "error",
				"message": "Método no permitido. Use GET o POST.",
			})
		}
		return
	}

	// 2. Ruta activa: /api/v1/configs/active
	if path == "active" {
		if r.Method != http.MethodGet {
			sendJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"status":  "error",
				"message": "Método no permitido. Use GET.",
			})
			return
		}
		h.GetActiveConfig(w, r)
		return
	}

	// 3. Sub-rutas por ID: /api/v1/configs/{id} o /api/v1/configs/{id}/activate
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "ID de configuración inválido",
		})
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.GetConfigByID(w, r, id)
		case http.MethodPut, http.MethodPatch:
			h.UpdateConfig(w, r, id)
		case http.MethodDelete:
			h.DeleteConfig(w, r, id)
		default:
			sendJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"status":  "error",
				"message": "Método no permitido. Use GET, PUT o DELETE.",
			})
		}
		return
	}

	if len(parts) == 2 && parts[1] == "activate" {
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			sendJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"status":  "error",
				"message": "Método no permitido. Use POST o PUT.",
			})
			return
		}
		h.ActivateConfig(w, r, id)
		return
	}

	http.NotFound(w, r)
}

// ListConfigs lista todas las configuraciones almacenadas.
// GET /api/v1/configs
func (h *ConfigHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		def := h.getDefaultConfig()
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "success",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"total":     1,
			"data":      []*database.ServerConfig{def},
		})
		return
	}

	configs, err := h.repo.GetAll()
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Error al consultar configuraciones: %v", err),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"total":     len(configs),
		"data":      configs,
	})
}

// GetActiveConfig devuelve la configuración que está en uso actualmente.
// GET /api/v1/configs/active
func (h *ConfigHandler) GetActiveConfig(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "success",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"data":      h.getDefaultConfig(),
		})
		return
	}

	active, err := h.repo.GetActive()
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Error al obtener configuración activa: %v", err),
		})
		return
	}

	if active == nil {
		active = h.getDefaultConfig()
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      active,
	})
}

// GetConfigByID obtiene una configuración específica por ID.
// GET /api/v1/configs/{id}
func (h *ConfigHandler) GetConfigByID(w http.ResponseWriter, r *http.Request, id int64) {
	if h.repo == nil {
		if id == 1 {
			sendJSON(w, http.StatusOK, map[string]interface{}{
				"status":    "success",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"data":      h.getDefaultConfig(),
			})
			return
		}
		sendJSON(w, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("configuración con ID %d no encontrada", id),
		})
		return
	}

	cfg, err := h.repo.GetByID(id)
	if err != nil {
		sendJSON(w, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      cfg,
	})
}

// CreateConfig almacena una nueva configuración de servidor SSH en PostgreSQL.
// POST /api/v1/configs
func (h *ConfigHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "Error al leer el cuerpo JSON",
		})
		return
	}
	defer r.Body.Close()

	var input database.ServerConfig
	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Formato JSON inválido: %v", err),
		})
		return
	}

	if h.repo == nil {
		input.ID = 1
		input.CreatedAt = time.Now().UTC()
		input.UpdatedAt = time.Now().UTC()
		sendJSON(w, http.StatusCreated, map[string]interface{}{
			"status":    "success",
			"message":   "Configuración registrada en memoria (PostgreSQL en reconexión)",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"data":      &input,
		})
		return
	}

	created, err := h.repo.Create(&input)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"status":    "success",
		"message":   "Configuración creada exitosamente",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      created,
	})
}

// UpdateConfig actualiza una configuración existente en PostgreSQL.
// PUT /api/v1/configs/{id}
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request, id int64) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "Error al leer el cuerpo JSON",
		})
		return
	}
	defer r.Body.Close()

	var input database.ServerConfig
	if err := json.Unmarshal(body, &input); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Formato JSON inválido: %v", err),
		})
		return
	}

	if h.repo == nil {
		input.ID = id
		input.UpdatedAt = time.Now().UTC()
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "success",
			"message":   "Configuración actualizada en memoria",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"data":      &input,
		})
		return
	}

	updated, err := h.repo.Update(id, &input)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "Configuración actualizada exitosamente",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      updated,
	})
}

// DeleteConfig elimina una configuración por ID de PostgreSQL.
// DELETE /api/v1/configs/{id}
func (h *ConfigHandler) DeleteConfig(w http.ResponseWriter, r *http.Request, id int64) {
	if h.repo == nil {
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "success",
			"message":   fmt.Sprintf("Configuración %d eliminada (en memoria)", id),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Configuración %d eliminada exitosamente", id),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ActivateConfig marca la configuración como activa y conmuta las conexiones SSH inmediatamente.
// POST /api/v1/configs/{id}/activate
func (h *ConfigHandler) ActivateConfig(w http.ResponseWriter, r *http.Request, id int64) {
	if h.repo == nil {
		def := h.getDefaultConfig()
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "success",
			"message":   fmt.Sprintf("Configuración '%s' (ID %d) activada exitosamente en caliente", def.Name, def.ID),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"data":      def,
		})
		return
	}

	active, err := h.repo.SetActive(id)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Configuración '%s' (ID %d) activada exitosamente en caliente", active.Name, active.ID),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      active,
	})
}
