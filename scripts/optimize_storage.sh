#!/bin/bash
# Optimize storage by hardlinking identical files in data/uploads to modules/

MODULES_DIR="modules"
UPLOADS_DIR="data/uploads"
COUNT=0
SAVED=0

echo "Optimizing storage..."

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
        
        # Check if contents match exactly
        if cmp -s "$src" "$potential"; then
            # Check if they are already the same inode (already hardlinked)
            inode_src=$(stat -c %i "$src")
            inode_pot=$(stat -c %i "$potential")
            
            if [ "$inode_src" != "$inode_pot" ]; then
                # Link them (force overwrites target with link to source)
                ln -f "$src" "$potential"
                
                # Verify link success
                if [ $? -eq 0 ]; then
                    # Calculate saved space
                    size=$(stat -c %s "$src")
                    SAVED=$((SAVED + size))
                    COUNT=$((COUNT + 1))
                    echo "🔗 Linked $(basename "$potential") -> $(basename "$src")"
                fi
            fi
        fi
    done
done

# Convert to readable size
if [ $SAVED -gt 1048576 ]; then
    SAVED_FMT="$((SAVED / 1048576)) MB"
elif [ $SAVED -gt 1024 ]; then
    SAVED_FMT="$((SAVED / 1024)) KB"
else
    SAVED_FMT="$SAVED bytes"
fi

echo ""
echo "✓ Optimization complete."
echo "  Files linked: $COUNT"
echo "  Space saved: $SAVED_FMT"
