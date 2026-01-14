# Module Consolidation Plan

## Current State
- **69 individual YAML files** (one module per file)
- Difficult to maintain and navigate
- Redundant metadata across related modules

## Proposed Consolidation

### Subject-Matter Groups

#### 1. **file_operations.yaml** (15 modules)
- check_file_exists
- check_is_directory
- check_path_exists
- check_permissions
- copy_file
- copy_directory
- copy_directory_v2
- create_file
- create_directory
- create_symlink
- delete_file
- move_file
- find_files
- find_large_files
- compare_files

#### 2. **file_utilities.yaml** (6 modules)
- archive_directory
- backup_directory
- cleanup_old_files
- extract_archive
- count_lines
- deduplicate_lines

#### 3. **file_content.yaml** (5 modules)
- get_file_info
- get_directory_size
- sort_lines
- search_in_files
- replace_in_files

#### 4. **system_info.yaml** (8 modules)
- detect_os
- get_system_info
- get_current_user
- get_current_directory
- list_directory
- list_processes
- show_environment_variables
- system_monitor

#### 5. **system_checks.yaml** (5 modules)
- check_command_exists
- check_disk_space
- check_network_connectivity
- check_port_status
- where_is_command

#### 6. **permissions.yaml** (3 modules)
- change_owner
- change_permissions
- set_sgid

#### 7. **network.yaml** (4 modules)
- download_file
- ping_host
- traceroute
- test_dns

#### 8. **docker.yaml** (3 modules)
- docker_install
- docker_container_manage
- docker_compose_manage

#### 9. **git.yaml** (2 modules)
- git_setup
- git_status_check

#### 10. **database.yaml** (3 modules)
- database_backup
- database_clients_install
- database_restore

#### 11. **web_servers.yaml** (3 modules)
- nginx_setup
- ssl_certificate_manager
- where_is_nginx_config

#### 12. **development_tools.yaml** (4 modules)
- dev_tools_install
- modern_cli_tools_install
- cloud_cli_tools_install
- network_security_tools_install

#### 13. **automation.yaml** (3 modules)
- cron_job_manager
- monitor_logs
- stop_process

#### 14. **termux.yaml** (4 modules)
- termux_setup
- termux_battery_info
- termux_camera
- termux_storage_info

#### 15. **shell_setup.yaml** (2 modules)
- zsh_setup
- vim_dev_setup

#### 16. **package_management.yaml** (1 module)
- setup_development_environment

---

## Benefits

1. **Easier Navigation**: 16 files instead of 69
2. **Better Organization**: Logical grouping by subject matter
3. **Simplified Maintenance**: Related modules in same file
4. **Clearer Intent**: File names indicate purpose
5. **Reduced Redundancy**: Share metadata where applicable

## Implementation Approach

### Option 1: Multiple Modules per File (Recommended)

```yaml
# file_operations.yaml
modules:
  - id: org.themobileprof.copy_file
    name: copy_file
    version: "1.0.0"
    description: Copy a file from source to destination
    tags: [file-ops, copy]
    flows:
      main:
        start: validate
        steps:
          # ...
  
  - id: org.themobileprof.delete_file
    name: delete_file
    version: "1.0.0"
    description: Delete a file safely
    tags: [file-ops, delete]
    flows:
      main:
        start: check_exists
        steps:
          # ...
```

**Pros:**
- Each module maintains independence
- Intent detection works per module
- Backwards compatible with CLI (`run copy_file`)
- Easy to share individual modules

**Cons:**
- Requires loader changes to support arrays

### Option 2: Single Module with Multiple Flows

```yaml
# file_operations.yaml
name: file_operations
id: org.themobileprof.file_operations
version: "1.0.0"
description: Common file operations
tags: [file-ops, filesystem]
flows:
  copy_file:
    start: validate
    steps: # ...
  
  delete_file:
    start: check_exists
    steps: # ...
```

**Pros:**
- No loader changes needed
- Simpler structure

**Cons:**
- Intent detection becomes harder (one module, many flows)
- Must always specify flow: `run file_operations copy_file`
- Tags/metadata shared across all operations
- Less granular versioning

---

## Recommended: Option 1

**Next Steps:**
1. Update `internal/modules/loader.go` to support YAML arrays
2. Update module model to support multi-module files
3. Create consolidated YAML files
4. Test intent detection still works
5. Update documentation

## Migration Strategy

### Phase 1: Add Multi-Module Support (Week 1)
- Update loader to detect and parse module arrays
- Maintain backwards compatibility with single-module files
- Add tests for multi-module loading

### Phase 2: Create Consolidated Files (Week 2)
- Create new consolidated YAML files in `modules/consolidated/`
- Keep old files for now (backwards compatibility)
- Update registry with both versions

### Phase 3: Deprecation (Week 3+)
- Mark old individual files as deprecated
- Update all documentation to use consolidated files
- Remove old files in next major version

### Phase 4: Optional Enhancement
- Add `clipilot module create` CLI to help users add modules
- Support both formats in registry uploads
- Add validation for multi-module files

---

## File Size Estimates

| Consolidated File | Modules | Est. Size |
|-------------------|---------|-----------|
| file_operations.yaml | 15 | 30-45 KB |
| file_utilities.yaml | 6 | 12-18 KB |
| system_info.yaml | 8 | 16-24 KB |
| development_tools.yaml | 4 | 8-12 KB |
| termux.yaml | 4 | 8-12 KB |

Total: ~100-150 KB (vs current ~200-300 KB individual files)

---

## Questions to Consider

1. **Should we version consolidated files separately?**
   - Option A: Each module has its own version (recommended)
   - Option B: File has master version, modules inherit

2. **How to handle module dependencies?**
   - Keep `requires`/`provides` per module
   - Loader validates dependencies within same file

3. **Registry uploads?**
   - Allow both single and multi-module uploads
   - Registry serves individual modules on demand

4. **CLI backwards compatibility?**
   - `run copy_file` should still work (by module ID)
   - `run file_operations` could show submenu

---

**Decision Required:** Should we proceed with Option 1 implementation?
