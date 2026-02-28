---
id: task-26
title: Create dev command wrapper for Docker Compose in web_app
status: Done
assignee:
  - '@claude'
created_date: '2025-11-04 04:52'
updated_date: '2025-11-04 04:59'
labels:
  - docker
  - devex
  - tooling
dependencies: []
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add a developer convenience command (script or justfile recipe) in web_app directory that automatically brings up Docker Compose services and runs commands inside the appropriate container. Should support parameter passthrough for flexibility (e.g., 'dev mix test', 'dev mix ecto.migrate', 'dev iex -S mix phx.server').
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Command brings up Docker Compose services if not already running
- [x] #2 Command runs provided arguments inside the web application container
- [x] #3 All parameters are passed through correctly to the container command
- [x] #4 Command works for common scenarios: mix commands, iex, shell access
- [x] #5 Exit codes from container commands are properly propagated
- [x] #6 Command provides clear usage instructions when run without arguments
- [x] #7 Works with both interactive (iex) and non-interactive commands
- [x] #8 Documentation added to DEVELOPMENT.md explaining usage
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Explore existing docker-compose setup and understand service names
2. Create a dev script that checks if services are running
3. Implement parameter passthrough to docker compose exec
4. Handle interactive vs non-interactive commands (tty detection)
5. Ensure proper exit code propagation
6. Add usage instructions for no-argument case
7. Test with common scenarios (mix commands, iex, shell)
8. Update DEVELOPMENT.md with usage examples
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Created a bash script `./dev` in the web_app directory that serves as a convenience wrapper for Docker Compose commands.

The script provides the following features:
- Automatically detects and starts Docker Compose services if not running
- Runs commands inside the Phoenix container with proper parameter passthrough
- Automatically detects interactive vs non-interactive commands (iex, sh, bash get -it flag, others get -T flag)
- Properly propagates exit codes from container commands using exec
- Displays helpful usage instructions when run without arguments
- Suppresses unnecessary docker compose output for cleaner UX

Updated DEVELOPMENT.md with comprehensive examples showing how to use the dev script for common tasks like running tests, migrations, formatting, and opening interactive shells.

The script makes the Docker-based development workflow much more convenient by eliminating the need to manually manage service state or type verbose docker compose exec commands.
<!-- SECTION:NOTES:END -->
