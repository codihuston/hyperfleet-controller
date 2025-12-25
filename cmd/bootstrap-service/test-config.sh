#!/bin/bash

# Load environment variables from .env if it exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Check required environment variables
if [ -z "$GH_PAT" ]; then
    echo "Error: GH_PAT environment variable is required"
    exit 1
fi

if [ -z "$OWNER" ]; then
    echo "Error: OWNER environment variable is required"
    exit 1
fi

if [ -z "$REPO" ]; then
    echo "Error: REPO environment variable is required"
    exit 1
fi

echo "Generating GitHub runner registration token..."

# Generate registration token using GitHub API
RESPONSE=$(curl -s -L \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GH_PAT" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$OWNER/$REPO/actions/runners/registration-token")

# Check if the request was successful
if [ $? -ne 0 ]; then
    echo "Error: Failed to generate registration token"
    exit 1
fi

# Extract token and expiration
TOKEN=$(echo "$RESPONSE" | jq -r '.token')
EXPIRES_AT=$(echo "$RESPONSE" | jq -r '.expires_at')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "Error: Failed to extract token from response"
    echo "Response: $RESPONSE"
    exit 1
fi

echo "Token generated successfully, expires at: $EXPIRES_AT"

# Generate unique runner name
RUNNER_NAME="test-runner-$(date +%s)"

# Create test configuration
cat > /tmp/runner-config.json << EOF
{
  "method": "runner-token",
  "platform": "github-actions",
  "runner_token": "$TOKEN",
  "registration_url": "https://github.com/$OWNER/$REPO",
  "runner_name": "$RUNNER_NAME",
  "labels": ["test", "docker", "ephemeral"],
  "expires_at": "$EXPIRES_AT",
  "runner": {
    "download_url": "",
    "install_path": "/opt/actions-runner",
    "work_dir": "/tmp/runner-work",
    "config_script": "config.sh",
    "run_script": "run.sh",
    "os": "linux",
    "arch": ""
  }
}
EOF

echo "Configuration written to /tmp/runner-config.json"
echo "Runner name: $RUNNER_NAME"