package sshclient

import (
	"strings"
	"testing"

	"mitfv2/config"
)

func TestSanitizeFilename(t *testing.T) {
	config.RemoteLogPath = "/var/log"
	client := NewSSHClient()

	tests := []struct {
		name      string
		input     string
		wantPath  string
		wantError bool
	}{
		{
			name:      "Valid log file",
			input:     "syslog",
			wantPath:  "/var/log/syslog",
			wantError: false,
		},
		{
			name:      "Valid nested log file",
			input:     "nginx/access.log",
			wantPath:  "/var/log/nginx/access.log",
			wantError: false,
		},
		{
			name:      "Path traversal with dot dot",
			input:     "../../etc/passwd",
			wantError: true,
		},
		{
			name:      "Absolute path escaping root",
			input:     "/etc/shadow",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.sanitizeFilename(tt.input)
			if (err != nil) != tt.wantError {
				t.Fatalf("sanitizeFilename(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
			if !tt.wantError && got != tt.wantPath {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.wantPath)
			}
		})
	}
}

func TestSanitizeSubPath(t *testing.T) {
	config.RemoteLogPath = "/var/log"
	client := NewSSHClient()

	tests := []struct {
		name      string
		subPath   string
		wantPath  string
		wantError bool
	}{
		{
			name:      "Empty subpath uses base",
			subPath:   "",
			wantPath:  "/var/log",
			wantError: false,
		},
		{
			name:      "Dot subpath uses base",
			subPath:   ".",
			wantPath:  "/var/log",
			wantError: false,
		},
		{
			name:      "Valid subfolder",
			subPath:   "nginx",
			wantPath:  "/var/log/nginx",
			wantError: false,
		},
		{
			name:      "Path traversal attempt",
			subPath:   "../etc",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.sanitizeSubPath(tt.subPath)
			if (err != nil) != tt.wantError {
				t.Fatalf("sanitizeSubPath(%q) error = %v, wantError %v", tt.subPath, err, tt.wantError)
			}
			if !tt.wantError && got != tt.wantPath {
				t.Errorf("sanitizeSubPath(%q) = %q, want %q", tt.subPath, got, tt.wantPath)
			}
		})
	}
}

func TestParseLsOutput(t *testing.T) {
	raw := `total 128
drwxr-xr-x  2 root root 4.0K Sep 23 21:00 .
drwxr-xr-x 13 root root 4.0K Sep 22 13:18 ..
-rw-r--r--  1 root root  12K Sep 23 22:15 syslog
drwxr-xr-x  2 root root 4.0K Sep 20 10:00 nginx
-rw-r-----  1 syslog adm 256K Sep 23 23:00 auth.log
`
	entries := ParseLsOutput(raw, "/var/log")
	if len(entries) != 3 {
		t.Fatalf("Se esperaban 3 entradas (excluyendo . y ..), obtenidas %d", len(entries))
	}

	// syslog
	if entries[0].Name != "syslog" || entries[0].IsDir != false || entries[0].Size != "12K" {
		t.Errorf("Entrada syslog inesperada: %+v", entries[0])
	}
	// nginx directory
	if entries[1].Name != "nginx" || entries[1].IsDir != true {
		t.Errorf("Entrada nginx inesperada: %+v", entries[1])
	}
	// auth.log
	if entries[2].Name != "auth.log" || entries[2].Owner != "syslog" {
		t.Errorf("Entrada auth.log inesperada: %+v", entries[2])
	}
}

func TestAuthMethodsEmpty(t *testing.T) {
	config.SSHPassword = ""
	config.SSHKeyPath = ""
	client := NewSSHClient()

	_, err := client.getAuthMethods()
	if err == nil {
		t.Fatal("Se esperaba error si no hay contraseña ni llave SSH configurada")
	}
	if !strings.Contains(err.Error(), "no se configuraron credenciales") {
		t.Errorf("Mensaje de error inesperado: %v", err)
	}
}
