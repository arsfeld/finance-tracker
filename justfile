#!/usr/bin/env just --justfile

# Default recipe to run when just is called without arguments
default: build

# Build the project
build:
    #!/usr/bin/env bash
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    mkdir -p bin
    devenv shell -- go build -ldflags="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" -o bin/finance_tracker ./src

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

# Run the web server
web: build
    #!/usr/bin/env bash
    ./bin/finance_tracker web

# Run the web server with verbose logging
web-verbose: build
    #!/usr/bin/env bash
    ./bin/finance_tracker web --environment development

# Run the web server on a specific port
web-port port="8080": build
    #!/usr/bin/env bash
    ./bin/finance_tracker web --port {{port}}

# Run the web server in production mode
web-prod: build
    #!/usr/bin/env bash
    ./bin/finance_tracker web --environment production --port 8080

# Run the web server with hot-reload (development)
web-dev:
    #!/usr/bin/env bash
    devenv shell -- air

# Run the web server with hot-reload on a specific port
web-dev-port port="8080":
    #!/usr/bin/env bash
    devenv shell -- air -c .air.toml -- web --environment development --port {{port}}

# Watch and rebuild on changes (without running)
watch:
    #!/usr/bin/env bash
    devenv shell -- air -c .air.toml -build.cmd "go build -o ./bin/finance_tracker ./src" -build.args_bin ""

# Install frontend dependencies
frontend-install:
    #!/usr/bin/env bash
    npm install

# Run frontend development server
frontend-dev:
    #!/usr/bin/env bash
    npm run dev

# Build frontend for production
frontend-build:
    #!/usr/bin/env bash
    npm run build

# Run full development stack (backend + frontend)
dev:
    #!/usr/bin/env bash
    echo "Starting frontend dev server..."
    npm run dev &
    FRONTEND_PID=$!
    echo "Starting backend in development mode..."
    ./bin/finance_tracker web --environment development
    # Kill frontend when backend exits
    kill $FRONTEND_PID 2>/dev/null || true

# Build everything for production
build-all: frontend-build build
