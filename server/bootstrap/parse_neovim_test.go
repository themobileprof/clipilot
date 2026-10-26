package bootstrap

import (
	"os"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

func TestParseNeovimModule(t *testing.T) {
	data, err := os.ReadFile("../../modules/neovim_setup.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var module moduleSeed
	if err := yaml.Unmarshal(data, &module); err != nil {
		t.Fatal(err)
	}
	if module.ID != "neovim_setup" || module.Version == "" {
		t.Fatalf("id=%q version=%q", module.ID, module.Version)
	}
	if !strings.Contains(string(data), "full = input .. site_rules") {
		t.Fatal("HTML generate prompt does not append the Pico rules")
	}
}
