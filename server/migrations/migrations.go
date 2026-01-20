package migrations

import (
	"embed"
)

//go:embed *.sql
var content embed.FS

// GetInitialSchema returns the SQL for the initial database schema
func GetInitialSchema() (string, error) {
	data, err := content.ReadFile("001_initial_schema.sql")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
