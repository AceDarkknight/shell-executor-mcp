#!/bin/bash

# Shell Executor MCP - Single Node Test Script
# This script tests the basic functionality of a single server node

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Shell Executor MCP - Single Node Test"
echo "=========================================="
echo ""

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Check if server binary exists
if [ ! -f "cmd/server/server.exe" ] && [ ! -f "cmd/server/server" ]; then
    print_info "Building server..."
    cd cmd/server
    go build -o server.exe main.go
    cd ../..
    print_success "Server built successfully"
else
    print_success "Server binary found"
fi

# Check if client binary exists
if [ ! -f "cmd/client/client.exe" ] && [ ! -f "cmd/client/client" ]; then
    print_info "Building client..."
    cd cmd/client
    go build -o client.exe main.go
    cd ../..
    print_success "Client built successfully"
else
    print_success "Client binary found"
fi

# Create test config files
print_info "Creating test configuration files..."

# Server config
cat > test_server_config.json << EOF
{
  "port": 8080,
  "node_name": "test-node-01",
  "peers": [],
  "cluster_token": "test-token-123",
  "security": {
    "blacklisted_commands": ["rm", "mkfs", "shutdown", "reboot"],
    "dangerous_args_regex": []
  }
}
EOF

# Client config
cat > test_client_config.json << EOF
{
  "servers": [
    {
      "name": "test-server",
      "url": "http://localhost:8080"
    }
  ]
}
EOF

print_success "Configuration files created"

# Start the server in the background
print_info "Starting server on port 8080..."
if [ -f "cmd/server/server.exe" ]; then
    cmd/server/server.exe test_server_config.json > server.log 2>&1 &
elif [ -f "cmd/server/server" ]; then
    cmd/server/server test_server_config.json > server.log 2>&1 &
fi

SERVER_PID=$!
print_success "Server started (PID: $SERVER_PID)"

# Wait for server to be ready
print_info "Waiting for server to be ready..."
sleep 3

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    print_error "Server failed to start. Check server.log for details."
    cat server.log
    exit 1
fi

print_success "Server is running"

# Run tests
echo ""
echo "=========================================="
echo "Running Tests"
echo "=========================================="
echo ""

# Test 1: Simple command (echo)
print_info "Test 1: Running 'echo Hello World'..."
echo "echo Hello World" | timeout 5 cmd/client/client.exe test_client_config.json > client_output.log 2>&1 || true

if grep -q "Hello World" client_output.log; then
    print_success "Test 1 PASSED: echo command executed successfully"
else
    print_error "Test 1 FAILED: echo command did not produce expected output"
    cat client_output.log
fi

# Test 2: Command with error
print_info "Test 2: Running 'exit 1' (should show error)..."
echo "exit 1" | timeout 5 cmd/client/client.exe test_client_config.json > client_output.log 2>&1 || true

if grep -q "failed\|error" client_output.log; then
    print_success "Test 2 PASSED: Error handling works correctly"
else
    print_error "Test 2 FAILED: Error not properly reported"
    cat client_output.log
fi

# Test 3: Security check (blacklisted command)
print_info "Test 3: Running 'rm -rf /' (should be blocked)..."
echo "rm -rf /" | timeout 5 cmd/client/client.exe test_client_config.json > client_output.log 2>&1 || true

if grep -q "security violation\|blacklisted" client_output.log; then
    print_success "Test 3 PASSED: Security guard blocked dangerous command"
else
    print_error "Test 3 FAILED: Security guard did not block the command"
    cat client_output.log
fi

# Test 4: Hostname command
print_info "Test 4: Running 'hostname'..."
echo "hostname" | timeout 5 cmd/client/client.exe test_client_config.json > client_output.log 2>&1 || true

if grep -q "test-node-01\|localhost\|hostname" client_output.log; then
    print_success "Test 4 PASSED: Hostname command executed"
else
    print_error "Test 4 FAILED: Hostname command did not work"
    cat client_output.log
fi

# Cleanup
echo ""
echo "=========================================="
echo "Cleanup"
echo "=========================================="

print_info "Stopping server..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
print_success "Server stopped"

# Clean up test files
rm -f test_server_config.json test_client_config.json server.log client_output.log

echo ""
echo "=========================================="
echo "Single Node Test Complete"
echo "=========================================="
