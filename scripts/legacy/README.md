# Legacy Files

This directory contains files that are no longer used in the CLIPilot server-only architecture but are kept for reference.

## Files

### install.sh
Original CLIPilot installation script that installed the monolithic CLI+server binary. 

**Status:** Deprecated  
**Reason:** CLIPilot is now server-only. Clio is the new client, and its install script is hosted at `clipilot.themobileprof.com/clio`

### uninstall.sh  
Uninstallation script for the old CLIPilot CLI binary.

**Status:** Deprecated  
**Reason:** No longer needed since CLIPilot doesn't install as a CLI tool

### test_commands.sh
Test script for the old command indexing feature in CLIPilot CLI.

**Status:** Deprecated  
**Reason:** Tested features that were part of the removed CLI component

### verify_queries.sh
Test script that ran natural language queries through the old CLIPilot CLI.

**Status:** Deprecated  
**Reason:** CLI functionality moved to Clio project

## History

**Date Archived:** February 28, 2026  
**Architecture Change:** CLIPilot split into server (registry) and client (Clio) components

See `docs/CLIO_MIGRATION_GUIDE.md` for details on the new architecture.
