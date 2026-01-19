package intent

import (
	"math"
	"regexp"
	"strings"
)

// TFIDFEngine computes TF-IDF similarity for command matching
type TFIDFEngine struct {
	documents []Document
	idf       map[string]float64
	vocab     map[string]int
	stopwords map[string]bool
}

// Document represents a searchable command document
type Document struct {
	ID          string
	Name        string
	Category    string
	Description string
	Keywords    []string
	UseCases    []string // NEW: specific use cases for intent matching
	Text        string
	TermFreq    map[string]float64
}

// NewTFIDFEngine creates a new TF-IDF engine
func NewTFIDFEngine() *TFIDFEngine {
	return &TFIDFEngine{
		documents: []Document{},
		idf:       make(map[string]float64),
		vocab:     make(map[string]int),
		stopwords: buildStopwords(),
	}
}

// buildStopwords creates a set of common words to filter
func buildStopwords() map[string]bool {
	words := []string{"is", "the", "a", "an", "what", "how", "to", "do", "does", "can", "will"}
	m := make(map[string]bool)
	for _, w := range words {
		m[w] = true
	}
	return m
}

// AddDocument indexes a document for search
func (e *TFIDFEngine) AddDocument(doc Document) {
	// Weight command name heavily by repeating it
	nameRepeated := strings.Repeat(doc.Name+" ", 3)

	// Include use cases for better intent matching
	useCaseText := ""
	if len(doc.UseCases) > 0 {
		useCaseText = strings.Join(doc.UseCases, " ")
	}

	// Combine all text fields with weighting
	doc.Text = strings.Join([]string{
		nameRepeated,
		doc.Description,
		strings.Join(doc.Keywords, " "),
		useCaseText,
	}, " ")

	// Compute term frequencies
	doc.TermFreq = e.computeTermFreq(doc.Text)

	// Update vocabulary
	for term := range doc.TermFreq {
		e.vocab[term]++
	}

	e.documents = append(e.documents, doc)
}

// BuildIndex computes IDF values after all documents are added
func (e *TFIDFEngine) BuildIndex() {
	numDocs := float64(len(e.documents))
	if numDocs == 0 {
		return
	}

	// Compute IDF for each term
	for term, docCount := range e.vocab {
		e.idf[term] = math.Log(numDocs / float64(docCount))
	}
}

// Search finds documents matching the query
func (e *TFIDFEngine) Search(query string, limit int) []ScoredDocument {
	// Extract intent signals from query
	intentBoosts := e.extractIntentSignals(query)

	// Compute query term frequencies
	queryTF := e.computeTermFreq(query)

	// Score all documents
	scores := make([]ScoredDocument, 0, len(e.documents))
	for _, doc := range e.documents {
		score := e.cosineSimilarity(queryTF, doc.TermFreq)

		// Apply intent-based boosting
		intentBoost := e.calculateIntentBoost(doc, intentBoosts)
		score = score * (1.0 + intentBoost)

		// Apply command name exact match boost
		if e.hasExactMatch(query, doc.Name) {
			score *= 1.5
		}

		if score > 0 {
			scores = append(scores, ScoredDocument{
				Document: doc,
				Score:    score,
			})
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Limit results
	if limit > 0 && len(scores) > limit {
		scores = scores[:limit]
	}

	return scores
}

// extractIntentSignals identifies key intent patterns in the query
func (e *TFIDFEngine) extractIntentSignals(query string) map[string]float64 {
	query = strings.ToLower(query)
	signals := make(map[string]float64)

	// Port-related intent
	if regexp.MustCompile(`\bport\b|\bports\b|\b:\d+\b|\b\d+\b`).MatchString(query) {
		signals["port"] = 2.0
		if regexp.MustCompile(`\bblocking\b|\busing\b|\blisten\b|\boccupied\b|\bcheck\b`).MatchString(query) {
			signals["port_check"] = 3.0
		}
	}

	// Network intent
	if regexp.MustCompile(`\bnetwork\b|\bconnection\b|\bsocket\b|\btcp\b|\budp\b`).MatchString(query) {
		signals["network"] = 1.5
	}

	// Process intent
	if regexp.MustCompile(`\bprocess\b|\bpid\b|\bkill\b|\brunning\b`).MatchString(query) {
		signals["process"] = 1.5
	}

	// File intent
	if regexp.MustCompile(`\bfile\b|\bfind\b|\bsearch\b|\blocate\b`).MatchString(query) {
		signals["file"] = 1.5
	}

	// Monitor/watch intent
	if regexp.MustCompile(`\bmonitor\b|\bwatch\b|\btrack\b`).MatchString(query) {
		signals["monitor"] = 1.3
	}

	return signals
}

// calculateIntentBoost applies intent-based scoring boost
func (e *TFIDFEngine) calculateIntentBoost(doc Document, intentSignals map[string]float64) float64 {
	boost := 0.0
	docText := strings.ToLower(doc.Text)
	category := strings.ToLower(doc.Category)

	for intent, weight := range intentSignals {
		switch intent {
		case "port", "port_check":
			// Boost commands specifically for port checking
			if strings.Contains(doc.Name, "ss") || strings.Contains(doc.Name, "lsof") ||
				strings.Contains(doc.Name, "netstat") || strings.Contains(docText, "port") {
				boost += weight
			}
			// Penalize unrelated commands
			if strings.Contains(doc.Name, "midi") || strings.Contains(doc.Name, "alsa") {
				boost -= 2.0
			}
		case "network":
			if strings.Contains(category, "network") || strings.Contains(docText, "network") {
				boost += weight * 0.5
			}
		case "process":
			if strings.Contains(docText, "process") || strings.Contains(docText, "pid") {
				boost += weight * 0.5
			}
		case "file":
			if strings.Contains(category, "file") || strings.Contains(docText, "file") {
				boost += weight * 0.5
			}
		}
	}

	return boost
}

// hasExactMatch checks if query contains exact command name
func (e *TFIDFEngine) hasExactMatch(query, commandName string) bool {
	query = strings.ToLower(query)
	commandName = strings.ToLower(commandName)

	// Check for exact word match
	words := strings.Fields(query)
	for _, word := range words {
		if word == commandName {
			return true
		}
	}
	return false
}

// computeTermFreq calculates normalized term frequencies
func (e *TFIDFEngine) computeTermFreq(text string) map[string]float64 {
	text = strings.ToLower(text)
	tokens := strings.Fields(text)

	// Filter stopwords and count raw frequencies
	counts := make(map[string]int)
	for _, token := range tokens {
		if !e.stopwords[token] {
			counts[token]++
		}
	}

	// Normalize by max frequency
	maxCount := 0
	for _, count := range counts {
		if count > maxCount {
			maxCount = count
		}
	}

	// Compute normalized TF
	tf := make(map[string]float64)
	if maxCount > 0 {
		for term, count := range counts {
			tf[term] = float64(count) / float64(maxCount)
		}
	}

	return tf
}

// cosineSimilarity computes cosine similarity with TF-IDF weighting
func (e *TFIDFEngine) cosineSimilarity(tf1, tf2 map[string]float64) float64 {
	var dotProduct, norm1, norm2 float64

	// Compute dot product and norms
	allTerms := make(map[string]bool)
	for term := range tf1 {
		allTerms[term] = true
	}
	for term := range tf2 {
		allTerms[term] = true
	}

	for term := range allTerms {
		idf := e.idf[term]
		if idf == 0 {
			idf = 1
		}

		tfidf1 := tf1[term] * idf
		tfidf2 := tf2[term] * idf

		dotProduct += tfidf1 * tfidf2
		norm1 += tfidf1 * tfidf1
		norm2 += tfidf2 * tfidf2
	}

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// ScoredDocument represents a document with its relevance score
type ScoredDocument struct {
	Document Document
	Score    float64
}
