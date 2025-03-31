#!/usr/bin/env just --justfile

# Default recipe to run when just is called without arguments
default: build

# Build the project
build:
    #!/usr/bin/env bash
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o bin/finance_tracker ./src

# Run the project
run: build
    #!/usr/bin/env bash
    ./bin/finance_tracker --force

# Clean build artifacts
clean:
    #!/usr/bin/env bash
    rm -rf bin/

# Run with verbose logging
run-verbose: build
    #!/usr/bin/env bash
    ./bin/finance_tracker --verbose --force

# Run with specific notification channels
run-notify notifications: build
    #!/usr/bin/env bash
    ./bin/finance_tracker --notifications {{notifications}} --force
