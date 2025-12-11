package intent

// TinyLLM provides local LLM-based intent classification
// This is a placeholder for future integration with GGML models
type TinyLLM struct {
	modelPath string
	loaded    bool
}

// NewTinyLLM creates a new tiny LLM classifier
func NewTinyLLM(modelPath string) *TinyLLM {
	return &TinyLLM{
		modelPath: modelPath,
		loaded:    false,
	}
}

// Load loads the LLM model into memory
func (llm *TinyLLM) Load() error {
	// TODO: Implement GGML model loading
	// This would use bindings like:
	// - github.com/go-skynet/go-llama.cpp
	// - Or direct CGO bindings to llama.cpp
	// - Or execute external binary

	// For now, return not implemented
	return nil
}

// Classify runs classification on input text
func (llm *TinyLLM) Classify(input string, candidates []string) (label string, confidence float64, err error) {
	// TODO: Implement classification
	// 1. Prepare prompt with input and candidate labels
	// 2. Run inference (timeout: 500ms)
	// 3. Parse output for label and confidence

	// Placeholder: not implemented
	return "", 0.0, nil
}

// Unload releases model from memory
func (llm *TinyLLM) Unload() error {
	llm.loaded = false
	return nil
}

// IsLoaded returns whether the model is loaded
func (llm *TinyLLM) IsLoaded() bool {
	return llm.loaded
}
