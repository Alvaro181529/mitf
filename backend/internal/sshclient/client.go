package sshclient

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mitfv2/config"

	"golang.org/x/crypto/ssh"
)

// ServerTarget contiene los parámetros de conexión SSH y ruta remota de un servidor específico.
type ServerTarget struct {
	SSHHost          string
	SSHPort          int
	SSHUser          string
	SSHPassword      string
	SSHKeyPath       string
	SSHKeyPassphrase string
	RemoteLogPath    string
}

// LogFileInfo representa la metadata básica de un archivo de log remoto.
type LogFileInfo struct {
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
	Size     string `json:"size"`
	ModTime  string `json:"mod_time"`
}

// DirectoryEntry representa un elemento listado (archivo o carpeta) en el servidor Linux.
type DirectoryEntry struct {
	Name        string `json:"name"`
	FullPath    string `json:"full_path"`
	Permissions string `json:"permissions"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
	Size        string `json:"size"`
	ModTime     string `json:"mod_time"`
	IsDir       bool   `json:"is_dir"`
}

// DirectoryListing contiene el resultado de listar una ruta remota (como ls -lah).
type DirectoryListing struct {
	RemotePath string           `json:"remote_path"`
	TotalItems int              `json:"total_items"`
	RawOutput  string           `json:"raw_output"`
	Entries    []DirectoryEntry `json:"entries"`
}

// SSHClient gestiona la conectividad SSH con el servidor Linux remoto.
type SSHClient struct {
	target *ServerTarget
}

// NewSSHClient crea una nueva instancia del cliente SSH utilizando la configuración global en config.
func NewSSHClient() *SSHClient {
	return &SSHClient{target: nil}
}

// NewSSHClientForTarget crea una instancia del cliente SSH apuntando a un servidor específico.
func NewSSHClientForTarget(target ServerTarget) *SSHClient {
	return &SSHClient{target: &target}
}

// GetTarget retorna los parámetros de conexión efectivos del cliente.
func (c *SSHClient) GetTarget() ServerTarget {
	if c != nil && c.target != nil {
		return *c.target
	}
	return ServerTarget{
		SSHHost:          config.SSHHost,
		SSHPort:          config.SSHPort,
		SSHUser:          config.SSHUser,
		SSHPassword:      config.SSHPassword,
		SSHKeyPath:       config.SSHKeyPath,
		SSHKeyPassphrase: config.SSHKeyPassphrase,
		RemoteLogPath:    config.RemoteLogPath,
	}
}

// getAuthMethods genera los métodos de autenticación basados en el objetivo actual.
func (c *SSHClient) getAuthMethods() ([]ssh.AuthMethod, error) {
	t := c.GetTarget()
	var authMethods []ssh.AuthMethod

	// 1. Autenticación con Llave Privada
	if t.SSHKeyPath != "" {
		keyBytes, err := os.ReadFile(t.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("error al leer la llave privada en %s: %w", t.SSHKeyPath, err)
		}

		var signer ssh.Signer
		if t.SSHKeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(t.SSHKeyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("error al procesar llave privada: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// 2. Autenticación con Contraseña
	if t.SSHPassword != "" {
		authMethods = append(authMethods, ssh.Password(t.SSHPassword))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no se configuraron credenciales SSH (SSH_PASSWORD o SSH_KEY_PATH)")
	}

	return authMethods, nil
}

// connect establece una nueva conexión con el servidor Linux.
func (c *SSHClient) connect() (*ssh.Client, error) {
	t := c.GetTarget()
	authMethods, err := c.getAuthMethods()
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            t.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	port := t.SSHPort
	if port <= 0 {
		port = 22
	}
	addr := fmt.Sprintf("%s:%d", t.SSHHost, port)
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar a %s: %w", addr, err)
	}

	return client, nil
}

// ExecuteCommand ejecuta un comando en el servidor remoto Linux y retorna la salida combinada.
func (c *SSHClient) ExecuteCommand(cmd string) (string, error) {
	client, err := c.connect()
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("error al abrir sesión SSH: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("error ejecutando comando '%s': %s (%w)", cmd, string(output), err)
	}

	return string(output), nil
}

// TestConnection verifica si la conexión SSH al servidor Linux es exitosa.
func (c *SSHClient) TestConnection() (string, error) {
	out, err := c.ExecuteCommand("uname -a && uptime")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// sanitizeFilename previene vulnerabilidades de Path Traversal (evita salir de RemoteLogPath).
func (c *SSHClient) sanitizeFilename(filename string) (string, error) {
	t := c.GetTarget()
	cleanName := filepath.Clean(filename)
	if strings.Contains(cleanName, "..") || strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, "\\") {
		return "", fmt.Errorf("nombre de archivo no válido o intento de path traversal: %s", filename)
	}
	return filepath.Join(t.RemoteLogPath, cleanName), nil
}

// sanitizeSubPath valida y limpia un subdirectorio opcional dentro de RemoteLogPath.
func (c *SSHClient) sanitizeSubPath(subPath string) (string, error) {
	t := c.GetTarget()
	if subPath == "" || subPath == "." {
		return t.RemoteLogPath, nil
	}
	clean := filepath.Clean(subPath)
	if strings.Contains(clean, "..") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "\\") {
		return "", fmt.Errorf("subruta no válida o intento de path traversal: %s", subPath)
	}
	return filepath.Join(t.RemoteLogPath, clean), nil
}

// ListDirectory ejecuta un 'ls -lah' en la ruta configurada en RemoteLogPath (o en un subpath).
func (c *SSHClient) ListDirectory(subPath string) (*DirectoryListing, error) {
	targetPath, err := c.sanitizeSubPath(subPath)
	if err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf(`ls -lah "%s"`, targetPath)
	raw, err := c.ExecuteCommand(cmd)
	if err != nil {
		return nil, err
	}

	entries := ParseLsOutput(raw, targetPath)

	return &DirectoryListing{
		RemotePath: targetPath,
		TotalItems: len(entries),
		RawOutput:  raw,
		Entries:    entries,
	}, nil
}

// ParseLsOutput procesa la salida en texto de 'ls -lah' y la convierte en una lista estructurada.
func ParseLsOutput(raw string, parentPath string) []DirectoryEntry {
	var entries []DirectoryEntry
	scanner := bufio.NewScanner(strings.NewReader(raw))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "total ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 9 {
			perms := fields[0]
			owner := fields[2]
			group := fields[3]
			size := fields[4]
			modTime := fmt.Sprintf("%s %s %s", fields[5], fields[6], fields[7])
			name := strings.Join(fields[8:], " ")

			if name == "." || name == ".." {
				continue
			}

			isDir := strings.HasPrefix(perms, "d")
			entries = append(entries, DirectoryEntry{
				Name:        name,
				FullPath:    filepath.Join(parentPath, name),
				Permissions: perms,
				Owner:       owner,
				Group:       group,
				Size:        size,
				ModTime:     modTime,
				IsDir:       isDir,
			})
		}
	}

	return entries
}

// ListLogFiles lista los archivos de log ubicados en la ruta remota especificada.
func (c *SSHClient) ListLogFiles() ([]LogFileInfo, error) {
	t := c.GetTarget()
	cmd := fmt.Sprintf(`find "%s" -maxdepth 2 -type f \( -name "*.log" -o -name "*log*" -o -name "*.txt" \) -printf "%%f\t%%p\t%%s\t%%TY-%%Tm-%%Td %%TH:%%TM:%%TS\n" 2>/dev/null || ls -la "%s"`, t.RemoteLogPath, t.RemoteLogPath)

	out, err := c.ExecuteCommand(cmd)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("error al listar archivos en %s: %w", t.RemoteLogPath, err)
	}

	var files []LogFileInfo
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 4 {
			files = append(files, LogFileInfo{
				Name:     parts[0],
				FullPath: parts[1],
				Size:     parts[2] + " bytes",
				ModTime:  parts[3],
			})
		} else {
			files = append(files, LogFileInfo{
				Name:     line,
				FullPath: filepath.Join(t.RemoteLogPath, line),
				Size:     "-",
				ModTime:  "-",
			})
		}
	}

	return files, nil
}

// ReadLogFile lee las últimas N líneas (o todo si lines <= 0) de un archivo específico en la ruta remota.
func (c *SSHClient) ReadLogFile(filename string, lines int) (string, error) {
	fullPath, err := c.sanitizeFilename(filename)
	if err != nil {
		return "", err
	}

	var cmd string
	if lines > 0 {
		cmd = fmt.Sprintf(`tail -n %d "%s"`, lines, fullPath)
	} else {
		cmd = fmt.Sprintf(`cat "%s"`, fullPath)
	}

	return c.ExecuteCommand(cmd)
}

// StreamLogFile ejecuta un `tail -f` en el archivo de log remoto y envía las líneas por un canal hasta que el contexto se cancele.
func (c *SSHClient) StreamLogFile(ctx context.Context, filename string, lines int, lineChan chan<- string) error {
	fullPath, err := c.sanitizeFilename(filename)
	if err != nil {
		return err
	}

	client, err := c.connect()
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("error al crear sesión SSH: %w", err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error al obtener stdout pipe: %w", err)
	}

	if lines <= 0 {
		lines = 50
	}
	cmd := fmt.Sprintf(`tail -n %d -f "%s"`, lines, fullPath)

	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("error al iniciar tail -f: %w", err)
	}

	go func() {
		<-ctx.Done()
		session.Close()
	}()

	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			lineChan <- strings.TrimRight(line, "\r\n")
		}
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				break
			}
			return err
		}
	}

	return nil
}
