#!/bin/bash

set -e

# Load environment variables from .env if it exists (look in repo root)
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

echo "=== HyperFleet Bootstrap Service Test ==="
echo "Platform: $(uname -m)"
echo "Docker platform: $(docker version --format '{{.Server.Arch}}')"

# Check if .env file exists and has required GitHub variables
if [ ! -f .env ]; then
    echo "Error: .env file not found. Please create one with:"
    echo "GH_PAT=your_github_pat_token"
    echo "OWNER=your_github_username_or_org"
    echo "REPO=your_repository_name"
    exit 1
fi

# Check required GitHub variables
MISSING_VARS=""
if [ -z "$GH_PAT" ]; then
    MISSING_VARS="$MISSING_VARS GH_PAT"
fi
if [ -z "$OWNER" ]; then
    MISSING_VARS="$MISSING_VARS OWNER"
fi
if [ -z "$REPO" ]; then
    MISSING_VARS="$MISSING_VARS REPO"
fi

if [ ! -z "$MISSING_VARS" ]; then
    echo "Error: Missing required environment variables:$MISSING_VARS"
    echo ""
    echo "Please add these to your .env file:"
    for var in $MISSING_VARS; do
        case $var in
            GH_PAT)
                echo "GH_PAT=your_github_personal_access_token"
                ;;
            OWNER)
                echo "OWNER=your_github_username_or_organization"
                ;;
            REPO)
                echo "REPO=your_repository_name"
                ;;
        esac
    done
    echo ""
    echo "Note: The GitHub PAT needs 'repo' scope for private repos or 'public_repo' for public repos"
    exit 1
fi

# Generate test configuration
echo "Generating test configuration..."
./cmd/bootstrap-service/test-config.sh

if [ ! -f /tmp/runner-config.json ]; then
    echo "Error: Failed to generate configuration"
    exit 1
fi

echo "Configuration generated successfully:"
cat /tmp/runner-config.json | jq .

# Build Docker image
echo "Building Docker image..."
docker build -f cmd/bootstrap-service/Dockerfile -t hyperfleet-bootstrap:test .

echo "Docker image built successfully"

# Ask user if they want to run the test
echo ""
echo "WARNING: This will:"
echo "1. Download the GitHub Actions runner (~100MB)"
echo "2. Register a new ephemeral runner with your GitHub repository: $OWNER/$REPO"
echo "3. Wait for a job to run (or timeout after a few minutes)"
echo "4. Clean up and remove the runner automatically"
echo ""
read -p "Do you want to proceed with the test? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Test cancelled"
    exit 0
fi

# Run the container
echo "Running bootstrap service test..."
echo "You can trigger a workflow in your repository to see the runner in action!"
echo ""

docker run --rm \
    -v /tmp/runner-config.json:/etc/hyperfleet/runner-config.json:ro \
    -v /var/run/docker.sock:/var/run/docker.sock \
    --privileged \
    hyperfleet-bootstrap:test \
    --config /etc/hyperfleet/runner-config.json

echo "Test completed!"