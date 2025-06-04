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

# Run development stack in tmux with split panes
tmux-dev:
    #!/usr/bin/env bash
    SESSION_NAME="finance-tracker-dev"
    
    # Check if tmux session already exists
    if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
        echo "Tmux session '$SESSION_NAME' already exists. Attaching..."
        tmux attach-session -t "$SESSION_NAME"
    else
        echo "Creating new tmux session '$SESSION_NAME'..."
        # Create new session with first pane running web-dev
        tmux new-session -d -s "$SESSION_NAME" -c "$(pwd)" "just web-dev"
        
        # Split horizontally and run frontend-dev in the new pane
        tmux split-window -h -t "$SESSION_NAME" -c "$(pwd)" "just frontend-dev"
        
        # Attach to the session
        tmux attach-session -t "$SESSION_NAME"
    fi

# Run development stack in zellij with split panes
zellij-dev:
    #!/usr/bin/env bash
    SESSION_NAME="finance-tracker-dev"
    LAYOUT_FILE="zellij-layout.kdl"
    
    echo "Starting Zellij session '$SESSION_NAME' with layout..."
    # Check if session exists and attach, otherwise create new one
    if zellij list-sessions | grep -q "$SESSION_NAME"; then
        echo "Attaching to existing session..."
        zellij attach "$SESSION_NAME"
    else
        echo "Creating new session with layout..."
        # Create session with specific name using the layout
        zellij --new-session-with-layout "$LAYOUT_FILE" --session "$SESSION_NAME"
    fi

claude:
    zellij a claude -c

# Build everything for production
build-all: frontend-build build
