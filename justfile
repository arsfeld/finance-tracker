#!/usr/bin/env just --justfile

# Default recipe to run when just is called without arguments
default: build

# Build the web server
build:
    #!/usr/bin/env bash
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o bin/finance_server ./cmd/server

# Build the legacy CLI
build-cli:
    #!/usr/bin/env bash
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o bin/finance_tracker ./src

# Run the web server
run: build
    #!/usr/bin/env bash
    ./bin/finance_server

# Run both Go server and Vite dev server for development
dev:
    #!/usr/bin/env bash
    echo "Starting Go server on :8080 and Vite dev server on :5173..."
    (cd web && npm run dev) &
    VITE_PID=$!
    go run ./cmd/server &
    GO_PID=$!
    trap "kill $VITE_PID $GO_PID 2>/dev/null" EXIT
    wait

# Build the frontend
build-web:
    #!/usr/bin/env bash
    cd web && npm run build

# Clean build artifacts
clean:
    #!/usr/bin/env bash
    rm -rf bin/ web/dist/

# Run the legacy CLI
run-cli: build-cli
    #!/usr/bin/env bash
    ./bin/finance_tracker --force

# Run the legacy CLI with verbose logging
run-verbose: build-cli
    #!/usr/bin/env bash
    ./bin/finance_tracker --verbose --force

# Run the legacy CLI with specific notification channels
run-notify notifications: build-cli
    #!/usr/bin/env bash
    ./bin/finance_tracker --notifications {{notifications}} --force
