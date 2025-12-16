#!/bin/bash
# Module Query Helper Script
# Query modules by tags from the CLIPilot registry
# Set REGISTRY_URL environment variable or it will fail

REGISTRY_URL="${REGISTRY_URL:-}"
COOKIES="${COOKIES:-/tmp/cookies.txt}"

if [ -z "$REGISTRY_URL" ]; then
    echo "Error: REGISTRY_URL environment variable is required"
    echo "Example: export REGISTRY_URL=http://localhost:8082"
    echo "         export REGISTRY_URL=https://registry.yourdomain.com"
    exit 1
fi

usage() {
    cat << EOF
CLIPilot Module Tag Query Tool

Usage: $0 <command> [options]

Commands:
  list-tags              List all unique tags in the registry
  atomic                 List all atomic modules
  composite              List all composite modules
  file-ops               List all file operation modules
  system-ops             List all system operation modules
  network-ops            List all network operation modules
  devops-ops             List all DevOps operation modules
  text-ops               List all text processing modules
  installer              List all installation/setup modules
  backup                 List all backup-related modules
  monitoring             List all monitoring modules
  security               List all security-related modules
  development            List all development environment modules
  query <tag1> [tag2..]  Query modules matching ALL specified tags

Examples:
  $0 atomic                    # All building blocks
  $0 file-ops                  # All file operation modules
  $0 installer                 # All installation modules
  $0 query devops-ops monitoring  # DevOps monitoring modules
  $0 query file-ops atomic     # Atomic file operations

Environment Variables:
  REGISTRY_URL    Registry server URL (required - no default)
  COOKIES         Cookie file for authentication (default: /tmp/cookies.txt)
EOF
    exit 1
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed."
    echo "Install with: sudo apt install jq"
    exit 1
fi

# Fetch all modules
fetch_modules() {
    curl -s -b "$COOKIES" "$REGISTRY_URL/api/modules"
}

# List all unique tags
list_tags() {
    fetch_modules | jq -r '.[].tags[]?' | sort -u
}

# Query modules by tags (AND logic - module must have ALL tags)
query_by_tags() {
    local tags=("$@")
    local jq_filter='.[]'
    
    for tag in "${tags[@]}"; do
        jq_filter="$jq_filter | select((.tags // []) | contains([\"$tag\"]))"
    done
    
    fetch_modules | jq "$jq_filter | {name, version, description, tags}"
}

# Count modules by tags
count_by_tags() {
    local tags=("$@")
    local jq_filter='.[]'
    
    for tag in "${tags[@]}"; do
        jq_filter="$jq_filter | select((.tags // []) | contains([\"$tag\"]))"
    done
    
    fetch_modules | jq "[$jq_filter]" | jq 'length'
}

# Show summary with counts
show_summary() {
    echo "CLIPilot Module Registry - Tag Summary"
    echo "======================================="
    echo ""
    echo "Complexity Levels:"
    echo "  Atomic:     $(count_by_tags atomic) modules"
    echo "  Composite:  $(count_by_tags composite) modules"
    echo ""
    echo "Functional Categories:"
    echo "  file-ops:    $(count_by_tags file-ops) modules"
    echo "  system-ops:  $(count_by_tags system-ops) modules"
    echo "  network-ops: $(count_by_tags network-ops) modules"
    echo "  devops-ops:  $(count_by_tags devops-ops) modules"
    echo "  text-ops:    $(count_by_tags text-ops) modules"
    echo ""
    echo "Domain Tags:"
    echo "  installer:   $(count_by_tags installer) modules"
    echo "  backup:      $(count_by_tags backup) modules"
    echo "  monitoring:  $(count_by_tags monitoring) modules"
    echo "  security:    $(count_by_tags security) modules"
    echo "  development: $(count_by_tags development) modules"
    echo ""
    echo "Total Modules: $(fetch_modules | jq 'length')"
}

# Main command dispatch
case "${1:-}" in
    list-tags)
        list_tags
        ;;
    atomic)
        query_by_tags atomic
        ;;
    composite)
        query_by_tags composite
        ;;
    file-ops)
        query_by_tags file-ops
        ;;
    system-ops)
        query_by_tags system-ops
        ;;
    network-ops)
        query_by_tags network-ops
        ;;
    devops-ops)
        query_by_tags devops-ops
        ;;
    text-ops)
        query_by_tags text-ops
        ;;
    installer)
        query_by_tags installer
        ;;
    backup)
        query_by_tags backup
        ;;
    monitoring)
        query_by_tags monitoring
        ;;
    security)
        query_by_tags security
        ;;
    development)
        query_by_tags development
        ;;
    query)
        shift
        if [ $# -eq 0 ]; then
            echo "Error: query command requires at least one tag"
            usage
        fi
        query_by_tags "$@"
        ;;
    summary)
        show_summary
        ;;
    ""|--help|-h)
        usage
        ;;
    *)
        echo "Error: Unknown command '$1'"
        echo ""
        usage
        ;;
esac
