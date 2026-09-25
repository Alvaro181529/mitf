package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

var configMu sync.RWMutex

// ============================================================================
// VARIABLES GLOBALES DE CONEXIÓN SSH Y RUTA DE LOGS
// ============================================================================
var (
	// SSHHost: Dirección IP o nombre de host del servidor Linux remoto
	SSHHost string = "127.0.0.1"

	// SSHPort: Puerto del servicio SSH (por defecto 22)
	SSHPort int = 22

	// SSHUser: Usuario con el que se autenticará en el servidor Linux
	SSHUser string = "root"

	// SSHPassword: Password del usuario (opcional si se usa llave SSH)
	SSHPassword string = ""

	// SSHKeyPath: Ruta del archivo de llave privada (ej: ~/.ssh/id_rsa o id_ed25519)
	SSHKeyPath string = ""

	// SSHKeyPassphrase: Clave de la llave privada si está cifrada
	SSHKeyPassphrase string = ""

	// RemoteLogPath: Ruta específica en el servidor Linux donde se ubican los archivos de logs
	RemoteLogPath string = "/var/log"

	// ServerPort: Puerto HTTP en el que correrá este backend en Go
	ServerPort string = ":8080"

	// ============================================================================
	// VARIABLES DE CONEXIÓN A POSTGRESQL
	// ============================================================================
	DBHost     string = "localhost"
	DBPort     int    = 5432
	DBUser     string = "postgres"
	DBPassword string = "postgres"
	DBName     string = "mitfv2_db"
	DBSSLMode  string = "disable"
)

// LoadConfig carga las variables de entorno desde un archivo .env si existe,
// buscando en el directorio actual y directorios superiores.
func LoadConfig() {
	configMu.Lock()
	defer configMu.Unlock()

	envFiles := []string{".env", "../.env", "../../.env"}
	loaded := false
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			loaded = true
			break
		}
	}
	if !loaded {
		log.Println("[INFO] Archivo .env no encontrado o no cargable, utilizando variables de entorno o valores por defecto")
	}

	if val := os.Getenv("SSH_HOST"); val != "" {
		SSHHost = val
	}
	if val := os.Getenv("SSH_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			SSHPort = p
		}
	}
	if val := os.Getenv("SSH_USER"); val != "" {
		SSHUser = val
	}
	if val := os.Getenv("SSH_PASSWORD"); val != "" {
		SSHPassword = val
	}
	if val := os.Getenv("SSH_KEY_PATH"); val != "" {
		SSHKeyPath = val
	}
	if val := os.Getenv("SSH_KEY_PASSPHRASE"); val != "" {
		SSHKeyPassphrase = val
	}
	if val := os.Getenv("REMOTE_LOG_PATH"); val != "" {
		RemoteLogPath = val
	}
	if val := os.Getenv("SERVER_PORT"); val != "" {
		ServerPort = val
	}

	// PostgreSQL
	if val := os.Getenv("DB_HOST"); val != "" {
		DBHost = val
	}
	if val := os.Getenv("DB_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			DBPort = p
		}
	}
	if val := os.Getenv("DB_USER"); val != "" {
		DBUser = val
	}
	if val := os.Getenv("DB_PASSWORD"); val != "" {
		DBPassword = val
	}
	if val := os.Getenv("DB_NAME"); val != "" {
		DBName = val
	}
	if val := os.Getenv("DB_SSLMODE"); val != "" {
		DBSSLMode = val
	}

	log.Printf("[CONFIG] Servidor SSH configurado: %s@%s:%d | Ruta remota de logs: %s", SSHUser, SSHHost, SSHPort, RemoteLogPath)
	log.Printf("[CONFIG] Base de datos PostgreSQL: %s@%s:%d/%s (sslmode=%s)", DBUser, DBHost, DBPort, DBName, DBSSLMode)
}

// GetDSN genera la cadena de conexión para PostgreSQL.
func GetDSN() string {
	configMu.RLock()
	defer configMu.RUnlock()

	if DBPassword != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", DBUser, DBPassword, DBHost, DBPort, DBName, DBSSLMode)
	}
	return fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", DBUser, DBHost, DBPort, DBName, DBSSLMode)
}

// UpdateSSHConfig actualiza dinámicamente las variables de configuración SSH en memoria.
func UpdateSSHConfig(host string, port int, user, password, keyPath, keyPass, remotePath string) {
	configMu.Lock()
	defer configMu.Unlock()

	if host != "" {
		SSHHost = host
	}
	if port > 0 {
		SSHPort = port
	}
	if user != "" {
		SSHUser = user
	}
	SSHPassword = password
	SSHKeyPath = keyPath
	SSHKeyPassphrase = keyPass
	if remotePath != "" {
		RemoteLogPath = remotePath
	}

	log.Printf("[CONFIG] Configuración SSH actualizada en tiempo real: %s@%s:%d | Ruta: %s", SSHUser, SSHHost, SSHPort, RemoteLogPath)
}
