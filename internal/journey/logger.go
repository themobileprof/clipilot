package journey

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/themobileprof/clipilot/pkg/models"
)

// Journey represents a search session
type Journey struct {
	Timestamp       time.Time     `json:"timestamp"`
	Query           string        `json:"query"`
	Steps           []Step        `json:"steps"`
	FinalCandidates []models.Candidate `json:"final_candidates,omitempty"`
	UserSelection   string        `json:"user_selection,omitempty"`
}

// Step represents a distinct search phase (e.g., hybrid, fts, remote)
type Step struct {
	Source     string  `json:"source"`     // "hybrid", "fts", "remote"
	Candidates int     `json:"candidates"` // count of candidates found in this step
	TopScore   float64 `json:"top_score"`  // highest score in this step
	DurationMs int64   `json:"duration_ms"` // time taken for this step
	Details    string  `json:"details,omitempty"`
}

// Logger handles writing journeys to file
type Logger struct {
	mu           sync.Mutex
	current      *Journey
	logFilePath  string
}

var instance *Logger
var once sync.Once

// GetLogger returns the singleton logger instance
func GetLogger() *Logger {
	once.Do(func() {
		home, _ := os.UserHomeDir()
		logPath := filepath.Join(home, ".clipilot", "client_journey.json")
		instance = &Logger{
			logFilePath: logPath,
		}
	})
	return instance
}

// StartNewJourney begins a new logging session
func (l *Logger) StartNewJourney(query string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.current = &Journey{
		Timestamp: time.Now(),
		Query:     query,
		Steps:     make([]Step, 0),
	}
}

// AddStep records a search step
func (l *Logger) AddStep(source string, candidates int, topScore float64, duration time.Duration, details string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.current == nil {
		return
	}

	l.current.Steps = append(l.current.Steps, Step{
		Source:     source,
		Candidates: candidates,
		TopScore:   topScore,
		DurationMs: duration.Milliseconds(),
		Details:    details,
	})
}

// SetFinalCandidates records the final merged results
func (l *Logger) SetFinalCandidates(candidates []models.Candidate) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.current == nil {
		return
	}
	
	// Copy to avoid race conditions if caller modifies
	// Limit to top 5 for brevity
	limit := 5
	if len(candidates) < limit {
		limit = len(candidates)
	}
	
	l.current.FinalCandidates = make([]models.Candidate, limit)
	copy(l.current.FinalCandidates, candidates[:limit])
}

// EndJourney finalizes the log and writes to file (append mode)
func (l *Logger) EndJourney(selection string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.current == nil {
		return
	}

	l.current.UserSelection = selection
	
	// Append to file (JSONL format for simplicity: one JSON object per line)
	f, err := os.OpenFile(l.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: Failed to write journey log: %v\n", err)
		return
	}
	defer f.Close()

	data, _ := json.Marshal(l.current)
	f.Write(data)
	f.WriteString("\n")

	l.current = nil // Reset
}
