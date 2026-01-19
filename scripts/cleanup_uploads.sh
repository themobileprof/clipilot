#!/bin/bash
# Remove duplicate versions of modules from data/uploads that exist in modules/

MODULES_DIR="modules"
UPLOADS_DIR="data/uploads"
COUNT=0
FREED=0

echo "Cleaning up duplicate uploads..."

# Ensure directories exist
if [ ! -d "$MODULES_DIR" ] || [ ! -d "$UPLOADS_DIR" ]; then
    echo "Error: Directory not found"
    exit 1
fi

for src in "$MODULES_DIR"/*.yaml; do
    [ -e "$src" ] || continue
    
    base=$(basename "$src" .yaml)
    
    # Look for candidates starting with base name in data/uploads
    # Pattern matches: name-version-timestamp.yaml
    for potential in "$UPLOADS_DIR"/"$base"-*.yaml; do
        [ -e "$potential" ] || continue
        
        echo "Removing duplicate: $(basename "$potential")"
        size=$(stat -c %s "$potential")
        FREED=$((FREED + size))
        rm "$potential"
        COUNT=$((COUNT + 1))
    done
done

# Convert to readable size
if [ $FREED -gt 1048576 ]; then
    FREED_FMT="$((FREED / 1048576)) MB"
elif [ $FREED -gt 1024 ]; then
    FREED_FMT="$((FREED / 1024)) KB"
else
    FREED_FMT="$FREED bytes"
fi

echo ""
echo "✓ Cleanup complete."
echo "  Files removed: $COUNT"
echo "  Space freed: $FREED_FMT"
