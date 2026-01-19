package intent

import (
	"regexp"
	"strings"
)

// TextNormalizer handles text normalization and intent extraction
type TextNormalizer struct {
	stopWords map[string]bool
	verbMap   map[string]string
}

// NewTextNormalizer creates a new text normalizer
func NewTextNormalizer() *TextNormalizer {
	return &TextNormalizer{
		stopWords: buildStopWords(),
		verbMap:   buildVerbIntentMap(),
	}
}

// Normalize transforms user input into canonical form
func (n *TextNormalizer) Normalize(input string) (normalized string, intent string) {
	// 1. Lowercase
	text := strings.ToLower(input)

	// 2. Remove punctuation except hyphens in words
	text = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(text, " ")

	// 3. Tokenize
	tokens := strings.Fields(text)

	// 4. Extract intent from verbs
	intent = n.extractIntent(tokens)

	// 5. Remove stop words
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !n.stopWords[token] && len(token) > 1 {
			filtered = append(filtered, token)
		}
	}

	// 6. Apply verb lemmatization (simple rules)
	lemmatized := n.lemmatizeVerbs(filtered)

	normalized = strings.Join(lemmatized, " ")
	return normalized, intent
}

// extractIntent identifies the primary intent from tokens
func (n *TextNormalizer) extractIntent(tokens []string) string {
	for _, token := range tokens {
		if intent, exists := n.verbMap[token]; exists {
			return intent
		}
	}
	return "find" // Default intent
}

// lemmatizeVerbs applies simple verb normalization rules
func (n *TextNormalizer) lemmatizeVerbs(tokens []string) []string {
	result := make([]string, len(tokens))
	for i, token := range tokens {
		// Simple suffix stripping for common verb forms
		if strings.HasSuffix(token, "ing") {
			token = strings.TrimSuffix(token, "ing")
			// Handle doubling (running -> run)
			if len(token) > 2 && token[len(token)-1] == token[len(token)-2] {
				token = token[:len(token)-1]
			}
		} else if strings.HasSuffix(token, "ed") {
			token = strings.TrimSuffix(token, "ed")
		} else if strings.HasSuffix(token, "s") && len(token) > 3 {
			token = strings.TrimSuffix(token, "s")
		}
		result[i] = token
	}
	return result
}

// buildStopWords returns common English stop words
func buildStopWords() map[string]bool {
	words := []string{
		"a", "an", "and", "are", "as", "at", "be", "by", "for",
		"from", "has", "he", "in", "is", "it", "its", "of", "on",
		"that", "the", "to", "was", "will", "with", "what", "when",
		"where", "which", "who", "why", "how", "can", "could", "would",
		"should", "may", "might", "must", "shall", "i", "me", "my",
		"you", "your", "am", "do", "does", "did", "have", "had",
	}
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

// buildVerbIntentMap maps verbs to canonical intents
func buildVerbIntentMap() map[string]string {
	return map[string]string{
		// Show intent
		"show":    "show",
		"list":    "show",
		"display": "show",
		"see":     "show",
		"print":   "show",
		"view":    "show",
		"get":     "show",

		// Find intent
		"find":   "find",
		"search": "find",
		"locate": "find",
		"lookup": "find",
		"check":  "find",
		"detect": "find",
		"which":  "find",

		// Kill intent
		"kill":      "kill",
		"stop":      "kill",
		"terminate": "kill",
		"end":       "kill",
		"close":     "kill",
		"shutdown":  "kill",

		// Monitor intent
		"monitor": "monitor",
		"watch":   "monitor",
		"track":   "monitor",
		"observe": "monitor",
		"tail":    "monitor",
		"follow":  "monitor",

		// Start intent
		"start":  "start",
		"run":    "start",
		"launch": "start",
		"open":   "start",
		"begin":  "start",

		// Install intent
		"install": "install",
		"setup":   "install",
		"add":     "install",

		// Remove intent
		"remove":    "remove",
		"delete":    "remove",
		"uninstall": "remove",
		"drop":      "remove",

		// Configure intent
		"configure": "configure",
		"set":       "configure",
		"change":    "configure",
		"update":    "configure",
		"modify":    "configure",

		// Test intent
		"test":   "test",
		"verify": "test",
		"ping":   "test",
		"probe":  "test",
	}
}
