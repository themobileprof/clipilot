# Command Enhancement Quickstart

## 🎯 What This Does

Enhances command descriptions using Google Gemini AI to make them more searchable.

**Before**: `ss - another utility to investigate sockets`  
**After**: `socket utility - check ports listening connections tcp udp network`

## 🚀 Quick Start

### 1. Get Gemini API Key
Visit: https://makersuite.google.com/app/apikey

### 2. Build the Tool
```bash
./scripts/build_enhancer.sh
```

### 3. Enhance Commands

**Single command**:
```bash
./bin/enhance --api-key=YOUR_KEY --command=ss
```

**Batch from file**:
```bash
# Create JSON file
cat > commands.json << 'EOF'
[
  {"name": "ss", "description": "utility to investigate sockets"},
  {"name": "lsof", "description": "list open files"},
  {"name": "netstat", "description": "print network connections"}
]
EOF

# Enhance (automatically skips already enhanced)
./bin/enhance --api-key=YOUR_KEY --batch=commands.json

# Force re-enhancement
./bin/enhance --api-key=YOUR_KEY --batch=commands.json --force
```

**Auto-process submissions**:
```bash
# Enhance top 10 unprocessed user submissions (skips already enhanced)
./bin/enhance --api-key=YOUR_KEY 10
```

## 📊 View Enhancement Status

**View statistics** (shows enhanced count, unprocessed submissions, etc.):
```bash
./scripts/enhancement_stats.sh
```
& Options

```bash
# Dry run (show what would be enhanced)
./bin/enhance --api-key=YOUR_KEY --command=ss --dry-run

# Check if command is already enhanced (will skip if yes)
./bin/enhance --api-key=YOUR_KEY --command=ss
# Output: ⏭️  Command 'ss' already enhanced (use --force to re-enhance)

# Force re-enhancement (updates to new version)
./bin/enhance --api-key=YOUR_KEY --command=ss --force
```

## 🎯 Smart Tracking

The enhancer **automatically tracks** which commands have been enhanced:

- ✅ **Skips already enhanced** - Saves API costs and time
- ✅ **Only new discoveries** - Submitted commands checked against `enhanced_commands` table
- ✅ **Version tracking** - Each re-enhancement increments version number
- ✅ **Force flag** - Use `--force` to override and re-enhance
- ✅ **Statistics** - Run `./scripts/enhancement_stats.sh` to see what's done
## 🔍 Test Locally

```bash
# Dry run (show what would be enhanced)
./bin/enhance --api-key=YOUR_KEY --command=ss --dry-run
```

## 💰 Cost

**Gemini 1.5 Flash** (recommended):
- ~$0.00007 per command
- 1000 commands = ~$0.07
- 10,000 commands = ~$0.70

Very affordable!

## 🎨 Enhancement Format

The tool generates:
- **Enhanced Description**: Adds 3-5 searchable keywords (max 100 chars)
- **Keywords**: 5-7 search terms users might use
- **Category**: networking, process, filesystem, system, development, security, general
- **Use Cases**: 3-5 common scenarios

**Conservative prompting ensures**:
- Short descriptions (no bloat)
- Accurate information (no hallucinations)
- Searchable keywords (better discoverability)

## 📝 Environment Variables

```bash
export GEMINI_API_KEY=your_key_here
export DATABASE_PATH=data/registry.db  # Optional
```

Then:
```bash
./bin/enhance --command=ss  # Uses env vars
```

## 🔗 Integration

After enhancing commands on the server:

1. **Users sync**: `clipilot` → `sync-commands`
2. **Download enhancements**: Client receives AI-enhanced descriptions
3. **Rebuild index**: `model refresh`
4. **Better search**: "port 8080" now finds `ss` correctly!

## 🛠️ Development

The tool is at `cmd/enhance/main.go`. Key features:

- **Gemini API integration**: Calls Google's AI for enhancement
- **Batch processing**: Handle multiple commands efficiently
- **Queue management**: Process user submissions automatically
- **Dry run mode**: Test without saving
- **Versioning**: Track enhancement iterations

## 📚 Documentation

See [docs/COMMAND_SYNC.md](../docs/COMMAND_SYNC.md) for full architecture.

---

**Status**: Ready to use (just add real Gemini API call)  
**TODO**: Replace mock enhancement in `enhanceCommandDescription()` with actual API
