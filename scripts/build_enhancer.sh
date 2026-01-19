#!/bin/bash
# Build the command enhancement tool

set -e

echo "Building CLIPilot Command Enhancer..."
cd "$(dirname "$0")/.."

go build -o bin/enhance ./cmd/enhance

echo "✓ Built: bin/enhance"
echo ""
echo "Usage:"
echo "  ./bin/enhance --api-key=YOUR_GEMINI_KEY --command=ss"
echo "  ./bin/enhance --api-key=YOUR_GEMINI_KEY --batch=commands.json"
echo "  ./bin/enhance --api-key=YOUR_GEMINI_KEY 10  # Process 10 submissions"
echo ""
echo "Get Gemini API key: https://makersuite.google.com/app/apikey"
