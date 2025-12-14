# CLIPilot Module Tag System

## Overview

All modules now use a structured tagging system to enable easy discovery and batch downloads of related modules. Tags are organized into three categories:

## Tag Categories

### 1. Functional Categories (Primary Operation Type)

Pick **1-2** functional tags that describe what the module does:

- **`file-ops`** - File and directory operations (read, write, copy, move, delete, archive, permissions)
- **`system-ops`** - System information, processes, users, environment, disk
- **`network-ops`** - Network connectivity, downloads, port checks
- **`devops-ops`** - Professional operations (monitoring, services, containers, logs)
- **`text-ops`** - Text processing and manipulation (grep, diff, sort, count)

### 2. Complexity Level (Always Include)

Every module must have exactly **one** of these:

- **`atomic`** - Single-purpose building blocks (35 modules)
- **`composite`** - Multi-step workflows (18 modules)

### 3. Domain/Use Case Tags (Optional)

Add domain-specific tags for composite modules:

- **`installer`** - Installation and setup modules
- **`backup`** - Backup operations
- **`monitoring`** - Health checks and system monitoring
- **`security`** - Security-related operations (SSL, permissions)
- **`development`** - Development environment setup
- **`web`** - Web server related
- **`database`** - Database operations
- **`git`** - Version control operations
- **`docker`** - Container operations

## Tag Distribution

### Atomic Modules by Functional Category (35 total)

**file-ops (22 atomic):**
- Validation: check_path_exists, check_is_directory, check_file_exists, check_permissions
- Basic ops: create_directory, create_file, delete_file, move_file, copy_file
- Info: get_directory_size, get_file_info, list_directory, get_current_directory, show_common_paths
- Advanced: find_files, archive_directory, extract_archive, change_permissions, change_owner, create_symlink
- Read: read_file

**system-ops (8 atomic):**
- Info: get_system_info, detect_os, check_disk_space, get_current_user, show_environment
- Validation: check_command_exists
- Process: list_processes, stop_process

**network-ops (2 atomic):**
- check_network_connectivity, download_file

**text-ops (5 atomic):**
- search_in_file, count_lines, compare_files, sort_lines, deduplicate_lines

**devops-ops (1 atomic):**
- check_port_status

### Composite Modules by Domain (18 total)

**installer (7 composites):**
- git_setup, docker_install, nginx_setup, nodejs_setup, python_dev_setup, vim_dev_setup, package_install

**file-ops workflows (5 composites):**
- copy_directory, copy_directory_v2, backup_directory, organize_downloads, cleanup_old_files, find_large_files

**devops-ops (7 composites):**
- monitor_logs, service_control, docker_container_manage, git_status_check, ssl_cert_check, database_backup, cron_job_manager

**system-ops (1 composite):**
- system_health_check

## Usage Examples

### Download All File Operation Modules

```bash
# Get all atomic file operations
curl -b cookies.txt "http://localhost:8080/api/modules" | \
  jq '.[] | select(.tags | contains(["file-ops", "atomic"]))'

# Get all file-related composites
curl -b cookies.txt "http://localhost:8080/api/modules" | \
  jq '.[] | select(.tags | contains(["file-ops", "composite"]))'
```

### Download All Installation/Setup Modules

```bash
curl -b cookies.txt "http://localhost:8080/api/modules" | \
  jq '.[] | select(.tags | contains(["installer"]))'
```

### Download All DevOps Operations

```bash
curl -b cookies.txt "http://localhost:8080/api/modules" | \
  jq '.[] | select(.tags | contains(["devops-ops"]))'
```

### Download All Atomic Modules (Building Blocks)

```bash
curl -b cookies.txt "http://localhost:8080/api/modules" | \
  jq '.[] | select(.tags | contains(["atomic"]))'
```

### Download Security-Related Modules

```bash
curl -b cookies.txt "http://localhost:8080/api/modules" | \
  jq '.[] | select(.tags | contains(["security"]))'
```

## Tag Combinations for Common Use Cases

### Server Administration Bundle
- **Tags:** `devops-ops`, `monitoring`, `service`
- **Modules:** monitor_logs, service_control, ssl_cert_check, cron_job_manager, system_health_check

### Development Environment Bundle
- **Tags:** `installer`, `development`
- **Modules:** git_setup, nodejs_setup, python_dev_setup, vim_dev_setup, package_install

### File Management Bundle
- **Tags:** `file-ops`, `atomic`
- **Modules:** All 22 atomic file operation modules

### Backup & Maintenance Bundle
- **Tags:** `backup`, `cleanup`, `maintenance`
- **Modules:** backup_directory, cleanup_old_files, organize_downloads, find_large_files

### Container Operations Bundle
- **Tags:** `docker`
- **Modules:** docker_install, docker_container_manage

### Text Processing Bundle
- **Tags:** `text-ops`
- **Modules:** search_in_file, count_lines, compare_files, sort_lines, deduplicate_lines

## Benefits

1. **Batch Downloads:** Download related modules together instead of one at a time
2. **Discoverability:** Find modules by purpose without knowing exact names
3. **Curated Bundles:** Create domain-specific module collections
4. **Granular Control:** Balance between "too broad" (download everything) and "too narrow" (one module at a time)
5. **Progressive Learning:** Start with atomic modules, move to composites

## Tag Design Principles

1. **Balanced Granularity:** Categories are neither too broad (all "utility") nor too narrow (separate tag per module)
2. **Multiple Dimensions:** Functional (what), Complexity (how), Domain (who/when)
3. **Composable:** Tags can be combined for precise queries
4. **Intuitive:** Tag names match common mental models and use cases
5. **Stable:** Tag structure is maintainable as library grows

## Future Enhancements

- Tag-based search in web UI
- Pre-defined module bundles (e.g., "DevOps Essentials")
- Tag-based module dependencies
- Download entire tag categories with one command
- Tag-based module recommendations
