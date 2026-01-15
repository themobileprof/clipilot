package ml

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// EmbeddingEngine provides semantic embeddings using ONNX Runtime
// with the all-MiniLM-L6-v2 model (INT8 quantized ~23MB)
type EmbeddingEngine struct {
	session       *ort.AdvancedSession
	tokenizer     *WordPieceTokenizer
	modelPath     string
	tokenizerPath string
	maxSeqLen     int
	embeddingDim  int
	loaded        bool
	mu            sync.RWMutex
}

// WordPieceTokenizer implements tokenization for BERT-style models
type WordPieceTokenizer struct {
	vocab      map[string]int32
	idToToken  map[int32]string
	unkTokenID int32
	padTokenID int32
	clsTokenID int32
	sepTokenID int32
	maxSeqLen  int
}

// NewEmbeddingEngine creates a new embedding engine
func NewEmbeddingEngine(modelDir string) *EmbeddingEngine {
	return &EmbeddingEngine{
		modelPath:     filepath.Join(modelDir, "model_quantized.onnx"),
		tokenizerPath: filepath.Join(modelDir, "vocab.txt"),
		maxSeqLen:     128, // Shorter for memory efficiency on Termux
		embeddingDim:  384, // all-MiniLM-L6-v2 dimension
		loaded:        false,
	}
}

// Load initializes the ONNX Runtime session and tokenizer
func (e *EmbeddingEngine) Load() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.loaded {
		return nil
	}

	// Check if model exists
	if _, err := os.Stat(e.modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model not found at %s - run 'clipilot model download' first", e.modelPath)
	}

	// Load tokenizer
	tokenizer, err := LoadWordPieceTokenizer(e.tokenizerPath, e.maxSeqLen)
	if err != nil {
		return fmt.Errorf("failed to load tokenizer: %w", err)
	}
	e.tokenizer = tokenizer

	// Initialize ONNX Runtime (use shared library)
	err = ort.InitializeEnvironment()
	if err != nil {
		return fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Create session options for efficiency (especially on Termux/low-RAM devices)
	options, err := ort.NewSessionOptions()
	if err != nil {
		return fmt.Errorf("failed to create session options: %w", err)
	}
	defer func() { _ = options.Destroy() }()

	// Set intra-op threads for better mobile performance
	if err := options.SetIntraOpNumThreads(2); err != nil {
		return fmt.Errorf("failed to set threads: %w", err)
	}

	// Create input/output shapes
	// all-MiniLM-L6-v2 inputs: input_ids, attention_mask, token_type_ids
	inputIDs, err := ort.NewTensor(ort.NewShape(1, int64(e.maxSeqLen)), make([]int64, e.maxSeqLen))
	if err != nil {
		return fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	attentionMask, err := ort.NewTensor(ort.NewShape(1, int64(e.maxSeqLen)), make([]int64, e.maxSeqLen))
	if err != nil {
		return fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	tokenTypeIDs, err := ort.NewTensor(ort.NewShape(1, int64(e.maxSeqLen)), make([]int64, e.maxSeqLen))
	if err != nil {
		return fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}

	// Output: sentence embeddings [1, 384]
	output, err := ort.NewTensor(ort.NewShape(1, int64(e.embeddingDim)), make([]float32, e.embeddingDim))
	if err != nil {
		return fmt.Errorf("failed to create output tensor: %w", err)
	}

	// Create advanced session
	session, err := ort.NewAdvancedSession(
		e.modelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"sentence_embedding"},
		[]ort.ArbitraryTensor{inputIDs, attentionMask, tokenTypeIDs},
		[]ort.ArbitraryTensor{output},
		options,
	)
	if err != nil {
		return fmt.Errorf("failed to create ONNX session: %w", err)
	}

	e.session = session
	e.loaded = true

	return nil
}

// Embed generates an embedding vector for the given text
func (e *EmbeddingEngine) Embed(text string) ([]float32, error) {
	e.mu.RLock()
	if !e.loaded {
		e.mu.RUnlock()
		return nil, fmt.Errorf("embedding engine not loaded")
	}
	e.mu.RUnlock()

	// Tokenize input
	inputIDsData, attentionMaskData, tokenTypeIDsData := e.tokenizer.Encode(text)

	// Create input tensors
	inputIDsTensor, err := ort.NewTensor(ort.NewShape(1, int64(e.maxSeqLen)), int64Slice(inputIDsData))
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	attentionMaskTensor, err := ort.NewTensor(ort.NewShape(1, int64(e.maxSeqLen)), int64Slice(attentionMaskData))
	if err != nil {
		_ = inputIDsTensor.Destroy()
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	tokenTypeIDsTensor, err := ort.NewTensor(ort.NewShape(1, int64(e.maxSeqLen)), int64Slice(tokenTypeIDsData))
	if err != nil {
		_ = inputIDsTensor.Destroy()
		_ = attentionMaskTensor.Destroy()
		return nil, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}

	// Output tensor
	outputData := make([]float32, e.embeddingDim)
	outputTensor, err := ort.NewTensor(ort.NewShape(1, int64(e.embeddingDim)), outputData)
	if err != nil {
		_ = inputIDsTensor.Destroy()
		_ = attentionMaskTensor.Destroy()
		_ = tokenTypeIDsTensor.Destroy()
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}

	// Run inference
	err = e.session.Run()
	if err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	// Clean up input tensors
	_ = inputIDsTensor.Destroy()
	_ = attentionMaskTensor.Destroy()
	_ = tokenTypeIDsTensor.Destroy()

	// Normalize embedding (L2 normalization for cosine similarity)
	normalized := normalizeL2(outputTensor.GetData())
	_ = outputTensor.Destroy()

	return normalized, nil
}

// EmbedBatch generates embeddings for multiple texts efficiently
func (e *EmbeddingEngine) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = emb
	}
	return results, nil
}

// CosineSimilarity computes cosine similarity between two embedding vectors
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	// Since vectors are L2-normalized, dot product = cosine similarity
	return dot
}

// Close releases ONNX Runtime resources
func (e *EmbeddingEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.session != nil {
		_ = e.session.Destroy()
		e.session = nil
	}
	e.loaded = false

	_ = ort.DestroyEnvironment()
	return nil
}

// IsLoaded returns whether the model is loaded
func (e *EmbeddingEngine) IsLoaded() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.loaded
}

// GetEmbeddingDim returns the embedding dimension
func (e *EmbeddingEngine) GetEmbeddingDim() int {
	return e.embeddingDim
}

// LoadWordPieceTokenizer loads vocabulary from vocab.txt
func LoadWordPieceTokenizer(vocabPath string, maxSeqLen int) (*WordPieceTokenizer, error) {
	data, err := os.ReadFile(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vocab file: %w", err)
	}

	vocab := make(map[string]int32)
	idToToken := make(map[int32]string)

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		token := strings.TrimSpace(line)
		if token == "" {
			continue
		}
		vocab[token] = int32(i)
		idToToken[int32(i)] = token
	}

	// Find special tokens
	unkID, ok := vocab["[UNK]"]
	if !ok {
		return nil, fmt.Errorf("vocab missing [UNK] token")
	}
	padID, ok := vocab["[PAD]"]
	if !ok {
		return nil, fmt.Errorf("vocab missing [PAD] token")
	}
	clsID, ok := vocab["[CLS]"]
	if !ok {
		return nil, fmt.Errorf("vocab missing [CLS] token")
	}
	sepID, ok := vocab["[SEP]"]
	if !ok {
		return nil, fmt.Errorf("vocab missing [SEP] token")
	}

	return &WordPieceTokenizer{
		vocab:      vocab,
		idToToken:  idToToken,
		unkTokenID: unkID,
		padTokenID: padID,
		clsTokenID: clsID,
		sepTokenID: sepID,
		maxSeqLen:  maxSeqLen,
	}, nil
}

// Encode tokenizes text and returns input_ids, attention_mask, token_type_ids
func (t *WordPieceTokenizer) Encode(text string) ([]int32, []int32, []int32) {
	// Normalize and tokenize
	tokens := t.tokenize(text)

	// Truncate if needed (reserve space for [CLS] and [SEP])
	maxTokens := t.maxSeqLen - 2
	if len(tokens) > maxTokens {
		tokens = tokens[:maxTokens]
	}

	// Build input_ids: [CLS] tokens... [SEP] [PAD]...
	inputIDs := make([]int32, t.maxSeqLen)
	attentionMask := make([]int32, t.maxSeqLen)
	tokenTypeIDs := make([]int32, t.maxSeqLen)

	inputIDs[0] = t.clsTokenID
	attentionMask[0] = 1

	for i, token := range tokens {
		id, ok := t.vocab[token]
		if !ok {
			id = t.unkTokenID
		}
		inputIDs[i+1] = id
		attentionMask[i+1] = 1
	}

	inputIDs[len(tokens)+1] = t.sepTokenID
	attentionMask[len(tokens)+1] = 1

	// Remaining positions are already 0 (PAD)
	return inputIDs, attentionMask, tokenTypeIDs
}

// tokenize performs WordPiece tokenization
func (t *WordPieceTokenizer) tokenize(text string) []string {
	// Lowercase and basic cleaning
	text = strings.ToLower(text)

	// Split on whitespace and punctuation
	var tokens []string
	wordRe := regexp.MustCompile(`[\w]+|[^\s\w]`)
	words := wordRe.FindAllString(text, -1)

	for _, word := range words {
		// Apply WordPiece algorithm
		subTokens := t.wordpiece(word)
		tokens = append(tokens, subTokens...)
	}

	return tokens
}

// wordpiece splits a word into WordPiece tokens
func (t *WordPieceTokenizer) wordpiece(word string) []string {
	if len(word) == 0 {
		return nil
	}

	// Check if whole word exists
	if _, ok := t.vocab[word]; ok {
		return []string{word}
	}

	var tokens []string
	start := 0

	for start < len(word) {
		end := len(word)
		var curToken string
		found := false

		for end > start {
			substr := word[start:end]
			if start > 0 {
				substr = "##" + substr
			}

			if _, ok := t.vocab[substr]; ok {
				curToken = substr
				found = true
				break
			}
			end--
		}

		if !found {
			// Single character not in vocab
			if start > 0 {
				tokens = append(tokens, "##"+string(word[start]))
			} else {
				tokens = append(tokens, string(word[start]))
			}
			start++
		} else {
			tokens = append(tokens, curToken)
			start = end
		}
	}

	return tokens
}

// Helper functions

func normalizeL2(vec []float32) []float32 {
	var sum float64
	for _, v := range vec {
		sum += float64(v * v)
	}
	norm := float32(math.Sqrt(sum))
	if norm == 0 {
		return vec
	}
	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}
	return result
}

func int64Slice(input []int32) []int64 {
	result := make([]int64, len(input))
	for i, v := range input {
		result[i] = int64(v)
	}
	return result
}

// EmbeddingCache stores pre-computed embeddings for fast lookup
type EmbeddingCache struct {
	embeddings map[string][]float32
	mu         sync.RWMutex
}

// NewEmbeddingCache creates a new embedding cache
func NewEmbeddingCache() *EmbeddingCache {
	return &EmbeddingCache{
		embeddings: make(map[string][]float32),
	}
}

// Get retrieves a cached embedding
func (c *EmbeddingCache) Get(key string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	emb, ok := c.embeddings[key]
	return emb, ok
}

// Set stores an embedding in the cache
func (c *EmbeddingCache) Set(key string, embedding []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.embeddings[key] = embedding
}

// Size returns the number of cached embeddings
func (c *EmbeddingCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.embeddings)
}

// Clear removes all cached embeddings
func (c *EmbeddingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.embeddings = make(map[string][]float32)
}

// ToJSON serializes the cache to JSON
func (c *EmbeddingCache) ToJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(c.embeddings)
}

// FromJSON loads the cache from JSON
func (c *EmbeddingCache) FromJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return json.Unmarshal(data, &c.embeddings)
}
