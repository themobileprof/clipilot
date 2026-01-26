package intent

import (
    "testing"

)

func TestDetectCopyFiles(t *testing.T) {
    database, cleanup := setupTestDB(t)
    defer cleanup()

    // Insert 'cp' command (what we want)
    _, err := database.Conn().Exec(`
        INSERT INTO commands (name, description, has_man)
        VALUES ('cp', 'copy files and directories', 1)
    `)
    if err != nil {
        t.Fatalf("Failed to insert cp command: %v", err)
    }

    // Insert 'unzip' command (what we don't want, but matches 'files')
    _, err = database.Conn().Exec(`
        INSERT INTO commands (name, description, has_man)
        VALUES ('unzip', 'list, test and extract compressed files in a ZIP archive', 1)
    `)
    if err != nil {
        t.Fatalf("Failed to insert unzip command: %v", err)
    }

    detector := NewDetector(database.Conn())

    // Test detection
    result, err := detector.Detect("how do I copy files")
    if err != nil {
        t.Fatalf("Detect failed: %v", err)
    }

    if result == nil || len(result.Candidates) == 0 {
        t.Fatal("Expected candidates")
    }

    top := result.Candidates[0]
    if top.Name != "cp" {
        t.Errorf("Expected top candidate to be 'cp', got '%s' (Score: %.2f)", top.Name, top.Score)
        for i, c := range result.Candidates {
            t.Logf("%d. %s (%.2f)", i+1, c.Name, c.Score)
        }
    }
}
