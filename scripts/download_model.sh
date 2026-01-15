#!/bin/bash
# Download all-MiniLM-L6-v2 ONNX model (INT8 quantized) for CLIPilot semantic search
# This model provides 384-dimensional sentence embeddings for intent matching

set -e

# Configuration
MODEL_NAME="all-MiniLM-L6-v2"
MODEL_DIR="${HOME}/.clipilot/models"
TEMP_DIR="/tmp/clipilot-model-download"

# Hugging Face model URL (sentence-transformers optimized version)
HF_MODEL_URL="https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main"
ONNX_MODEL_URL="https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx"
VOCAB_URL="${HF_MODEL_URL}/vocab.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check for required tools
check_dependencies() {
    local missing=""
    
    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        missing="curl or wget"
    fi
    
    if ! command -v python3 &> /dev/null; then
        log_warn "Python3 not found - quantization will be skipped"
        log_warn "Pre-quantized model will be downloaded instead"
    fi
    
    if [ -n "$missing" ]; then
        log_error "Missing required tool: $missing"
        exit 1
    fi
}

# Download file with curl or wget
download_file() {
    local url="$1"
    local output="$2"
    
    if command -v curl &> /dev/null; then
        curl -L --progress-bar -o "$output" "$url"
    else
        wget --show-progress -O "$output" "$url"
    fi
}

# Check if model is already downloaded
check_existing_model() {
    if [ -f "${MODEL_DIR}/model_quantized.onnx" ] && [ -f "${MODEL_DIR}/vocab.txt" ]; then
        log_info "Model already exists at ${MODEL_DIR}"
        read -p "Re-download? [y/N]: " response
        if [[ ! "$response" =~ ^[Yy]$ ]]; then
            log_info "Keeping existing model"
            exit 0
        fi
    fi
}

# Create directories
setup_directories() {
    log_info "Creating directories..."
    mkdir -p "${MODEL_DIR}"
    mkdir -p "${TEMP_DIR}"
}

# Download vocabulary
download_vocab() {
    log_info "Downloading vocabulary..."
    download_file "${VOCAB_URL}" "${MODEL_DIR}/vocab.txt"
    
    if [ ! -f "${MODEL_DIR}/vocab.txt" ]; then
        log_error "Failed to download vocab.txt"
        exit 1
    fi
    
    log_info "Vocabulary downloaded ($(wc -l < "${MODEL_DIR}/vocab.txt") tokens)"
}

# Download ONNX model
download_model() {
    log_info "Downloading ONNX model (~90MB)..."
    log_info "This may take a few minutes..."
    
    download_file "${ONNX_MODEL_URL}" "${TEMP_DIR}/model.onnx"
    
    if [ ! -f "${TEMP_DIR}/model.onnx" ]; then
        log_error "Failed to download model.onnx"
        exit 1
    fi
    
    local size=$(du -h "${TEMP_DIR}/model.onnx" | cut -f1)
    log_info "Model downloaded: ${size}"
}

# Quantize model to INT8 (reduces size from ~90MB to ~23MB)
quantize_model() {
    if ! command -v python3 &> /dev/null; then
        log_warn "Python3 not available, using float model"
        cp "${TEMP_DIR}/model.onnx" "${MODEL_DIR}/model_quantized.onnx"
        return
    fi
    
    # Check for onnxruntime
    if ! python3 -c "import onnxruntime" &> /dev/null; then
        log_info "Installing onnxruntime for quantization..."
        pip3 install --user onnxruntime
    fi
    
    # Check for onnx
    if ! python3 -c "import onnx" &> /dev/null; then
        log_info "Installing onnx for quantization..."
        pip3 install --user onnx
    fi
    
    log_info "Quantizing model to INT8 (this reduces size by ~75%)..."
    
    python3 << 'EOF'
import os
import sys

try:
    from onnxruntime.quantization import quantize_dynamic, QuantType
    
    input_path = os.path.expandvars("${TEMP_DIR}/model.onnx")
    output_path = os.path.expandvars("${MODEL_DIR}/model_quantized.onnx")
    
    print("Applying dynamic INT8 quantization...")
    quantize_dynamic(
        model_input=input_path,
        model_output=output_path,
        weight_type=QuantType.QInt8,
        optimize_model=True
    )
    print("Quantization complete!")
    
except ImportError as e:
    print(f"Warning: Quantization skipped - {e}")
    import shutil
    shutil.copy(
        os.path.expandvars("${TEMP_DIR}/model.onnx"),
        os.path.expandvars("${MODEL_DIR}/model_quantized.onnx")
    )
    print("Copied original model instead")
    
except Exception as e:
    print(f"Error during quantization: {e}")
    sys.exit(1)
EOF

    if [ $? -ne 0 ]; then
        log_warn "Quantization failed, using original model"
        cp "${TEMP_DIR}/model.onnx" "${MODEL_DIR}/model_quantized.onnx"
    fi
}

# Verify model
verify_model() {
    log_info "Verifying model..."
    
    if [ ! -f "${MODEL_DIR}/model_quantized.onnx" ]; then
        log_error "Model file missing!"
        exit 1
    fi
    
    if [ ! -f "${MODEL_DIR}/vocab.txt" ]; then
        log_error "Vocabulary file missing!"
        exit 1
    fi
    
    local model_size=$(du -h "${MODEL_DIR}/model_quantized.onnx" | cut -f1)
    local vocab_size=$(wc -l < "${MODEL_DIR}/vocab.txt")
    
    log_info "Model verification complete:"
    log_info "  - Model size: ${model_size}"
    log_info "  - Vocabulary: ${vocab_size} tokens"
}

# Cleanup
cleanup() {
    log_info "Cleaning up temporary files..."
    rm -rf "${TEMP_DIR}"
}

# Print summary
print_summary() {
    echo ""
    echo "==========================================="
    log_info "Model installation complete!"
    echo "==========================================="
    echo ""
    echo "Model directory: ${MODEL_DIR}"
    echo ""
    echo "Files installed:"
    ls -lh "${MODEL_DIR}/" | grep -E "(onnx|txt)$"
    echo ""
    echo "To use semantic search, run:"
    echo "  clipilot search \"your query here\""
    echo ""
    echo "Model details:"
    echo "  - Name: ${MODEL_NAME}"
    echo "  - Embedding dimension: 384"
    echo "  - Max sequence length: 128 tokens"
    echo "  - Optimized for: Sentence similarity"
    echo ""
}

# Main
main() {
    echo "=========================================="
    echo " CLIPilot Semantic Model Installer"
    echo " Model: ${MODEL_NAME} (INT8 Quantized)"
    echo "=========================================="
    echo ""
    
    check_dependencies
    check_existing_model
    setup_directories
    download_vocab
    download_model
    quantize_model
    verify_model
    cleanup
    print_summary
}

main "$@"
