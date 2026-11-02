package handlers

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const clioInstallScriptURL = "https://raw.githubusercontent.com/themobileprof/clio/main/install.sh"

var versionLineRE = regexp.MustCompile(`(?m)^VERSION=["']?([^"'\n]+)["']?`)

// EnsureClioInstallScript keeps /clio on the script from the Clio git main branch.
// An existing active script is replaced when that file changes.
func EnsureClioInstallScript(db *sql.DB, uploadsDir string) error {
	content, version, err := fetchClioInstallScript()
	if err != nil {
		return err
	}

	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)

	var currentPath, currentSum string
	err = db.QueryRow(`
		SELECT file_path, checksum_sha256 FROM install_scripts
		WHERE is_active = 1
		ORDER BY uploaded_at DESC
		LIMIT 1
	`).Scan(&currentPath, &currentSum)
	if err == nil && currentSum == checksum {
		if _, statErr := os.Stat(currentPath); statErr == nil {
			return nil
		}
	} else if err != nil && err != sql.ErrNoRows {
		return err
	}

	installDir := filepath.Join(uploadsDir, "install_scripts")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("create install_scripts dir: %w", err)
	}

	filename := fmt.Sprintf("install-%s-%s.sh", version, checksum[:8])
	filePath := filepath.Join(installDir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("write install script: %w", err)
	}

	if currentSum != "" {
		_, err = db.Exec(`
			UPDATE install_scripts
			SET version = ?, file_path = ?, checksum_sha256 = ?, size_bytes = ?, uploaded_at = CURRENT_TIMESTAMP
			WHERE is_active = 1
		`, version, filePath, checksum, len(content))
		if err != nil {
			_ = os.Remove(filePath)
			return fmt.Errorf("update install script: %w", err)
		}
		log.Printf("Bootstrap: Clio install script updated (%s)", version)
		return nil
	}

	var owner int64
	if err := db.QueryRow(`SELECT id FROM users WHERE role = 'admin' ORDER BY id LIMIT 1`).Scan(&owner); err != nil {
		_ = os.Remove(filePath)
		return fmt.Errorf("no admin user to own the install script: %w", err)
	}
	_, err = db.Exec(`
		INSERT INTO install_scripts (version, file_path, checksum_sha256, size_bytes, uploaded_by, is_active, uploaded_at)
		VALUES (?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP)
	`, version, filePath, checksum, len(content), owner)
	if err != nil {
		_ = os.Remove(filePath)
		return fmt.Errorf("insert install script record: %w", err)
	}
	log.Printf("Bootstrap: Clio install script ready (%s)", version)
	return nil
}

func fetchClioInstallScript() ([]byte, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(clioInstallScriptURL)
	if err != nil {
		return nil, "", fmt.Errorf("fetch install script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetch install script: HTTP %d", resp.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, "", fmt.Errorf("read install script: %w", err)
	}

	if !strings.Contains(string(content), "#!/") {
		return nil, "", fmt.Errorf("install script missing shebang")
	}

	version := "auto"
	if match := versionLineRE.FindSubmatch(content); len(match) > 1 {
		version = string(match[1])
	}

	return content, version, nil
}
