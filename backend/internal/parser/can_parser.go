package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ServerInfo representa la información del host remoto en la respuesta JSON.
type CANServerInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	User string `json:"user"`
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// ServerPayload modela la estructura recibida del endpoint /api/v1/logs/jedi-can.
type ServerPayload struct {
	Content    string        `json:"content"`
	File       string        `json:"file"`
	Lines      int           `json:"lines"`
	ServerInfo CANServerInfo `json:"server_info"`
	Status     string        `json:"status"`
	Timestamp  string        `json:"timestamp"`
}

// CANFrame modela una trama individual procesada del bus CAN (jedi_can_at_error.log).
type CANFrame struct {
	TimestampMS int64    `json:"timestamp_ms"` // Marca de tiempo relativa en ms
	Direction   string   `json:"direction"`    // "T" para Transmitido, "R" para Recibido
	CANID       string   `json:"can_id"`       // Identificador hexadecimal, ej. "478h", "208h"
	DLC         int      `json:"dlc"`          // Data Length Code (cantidad de bytes de datos)
	Payload     []string `json:"payload"`      // Lista de bytes hexadecimales del mensaje
	IsHeartbeat bool     `json:"is_heartbeat"` // true si el ID es "478h" o "479h"
	SubsystemID int      `json:"subsystem_id"` // Asignado a 1: Generador y Tubo RX
}

// CANParsedResult contiene el resultado consolidado del parseo de tramas CAN.
type CANParsedResult struct {
	File          string        `json:"file"`
	Timestamp     string        `json:"timestamp"`
	ServerInfo    CANServerInfo `json:"server_info"`
	SubsystemID   int           `json:"subsystem_id"` // Subsistema 1 (Generador y Tubo RX)
	SubsystemName string        `json:"subsystem_name"`
	TotalFrames   int           `json:"total_frames"`
	Frames        []CANFrame    `json:"frames"`
}

// Regex precompilada para extraer los campos de cada línea de log CAN.
// Ejemplo: " 5032265 T : 478h (0) "
// Ejemplo: " 5032130 R : 479h (1) 00h "
// Ejemplo: " 4940280 R : 208h (8) 01h 02h 03h 04h 05h 06h 07h 08h "
var reCANLine = regexp.MustCompile(`^\s*(\d+)\s+([TRtr])\s*:\s*([0-9a-fA-F]+h?)\s*\(\s*(\d+)\s*\)(.*)$`)

// ParseCANContent procesa el texto en crudo de jedi_can_at_error.log y retorna el listado de CANFrame.
func ParseCANContent(content string) []CANFrame {
	lines := strings.Split(content, "\n")
	frames := make([]CANFrame, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Descartar líneas vacías o separadores (como "-----------------------")
		if trimmed == "" || strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "===") {
			continue
		}

		matches := reCANLine.FindStringSubmatch(line)
		if len(matches) < 5 {
			continue
		}

		// 1. TimestampMS
		tsMS, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			continue
		}

		// 2. Dirección ("T" o "R")
		dir := strings.ToUpper(matches[2])

		// 3. CANID
		canID := strings.ToLower(matches[3])
		if !strings.HasSuffix(canID, "h") {
			canID = canID + "h"
		}

		// 4. DLC
		dlc, _ := strconv.Atoi(matches[4])

		// 5. Payload de bytes
		payloadStr := strings.TrimSpace(matches[5])
		var payload []string
		if payloadStr != "" {
			parts := strings.Fields(payloadStr)
			for _, p := range parts {
				trimmedByte := strings.TrimSpace(p)
				if trimmedByte != "" {
					payload = append(payload, trimmedByte)
				}
			}
		}
		if payload == nil {
			payload = []string{}
		}

		// 6. IsHeartbeat (478h o 479h)
		isHeartbeat := (canID == "478h" || canID == "479h")

		frames = append(frames, CANFrame{
			TimestampMS: tsMS,
			Direction:   dir,
			CANID:       canID,
			DLC:         dlc,
			Payload:     payload,
			IsHeartbeat: isHeartbeat,
			SubsystemID: 1, // Subsistema 1: Generador y Tubo RX
		})
	}

	return frames
}

// ParseCANServerPayload deserializa el JSON recibido y procesa las tramas del log CAN.
func ParseCANServerPayload(payloadJSON []byte) (*CANParsedResult, error) {
	var payload ServerPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, fmt.Errorf("error al deserializar payload JSON de CAN: %w", err)
	}

	frames := ParseCANContent(payload.Content)

	return &CANParsedResult{
		File:          payload.File,
		Timestamp:     payload.Timestamp,
		ServerInfo:    payload.ServerInfo,
		SubsystemID:   1, // Subsistema 1: Generador y Tubo RX
		SubsystemName: "Generador y Tubo RX (Subsystem 1)",
		TotalFrames:   len(frames),
		Frames:        frames,
	}, nil
}
