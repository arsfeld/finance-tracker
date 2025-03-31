#!/usr/bin/env just --justfile

# Default recipe to run when just is called without arguments
default: build

# Build the project
build:
    #!/usr/bin/env bash
    go build -o bin/finance_tracker ./src

# Run the project
run: build
    #!/usr/bin/env bash
    ./bin/finance_tracker

# Clean build artifacts
clean:
    #!/usr/bin/env bash
    rm -rf bin/

# Run with verbose logging
run-verbose: build
    #!/usr/bin/env bash
    ./bin/finance_tracker --verbose

# Run with specific notification channels
run-notify notifications: build
    #!/usr/bin/env bash
    ./bin/finance_tracker --notifications {{notifications}}
