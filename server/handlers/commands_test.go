package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSemanticSearchCatalog(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	handler := HandleSemanticSearch(db, "")

	body := `{"query":"check disk space on my phone","os":"linux","arch":"arm64"}`
	req := httptest.NewRequest(http.MethodPost, "/api/commands/search", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}

	var resp SemanticSearchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Candidates) == 0 {
		t.Fatal("no candidates")
	}
	if resp.Candidates[0].Name != "df" && resp.Candidates[0].Name != "du" {
		t.Fatalf("got %q", resp.Candidates[0].Name)
	}
	if resp.Source != "catalog" {
		t.Fatalf("source = %q", resp.Source)
	}
	if len(resp.Results) != len(resp.Candidates) {
		t.Fatal("legacy results alias missing")
	}
}
