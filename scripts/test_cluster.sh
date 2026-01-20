#!/bin/bash

# Shell Executor MCP - Cluster Test Script
# This script tests the cluster functionality with multiple server nodes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Shell Executor MCP - Cluster Test"
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

# Create test config files for 3 nodes
print_info "Creating test configuration files for 3 nodes..."

# Node 1 config (primary node)
cat > test_node1_config.json << EOF
{
  "port": 8080,
  "node_name": "node-01",
  "peers": ["http://localhost:8081", "http://localhost:8082"],
  "cluster_token": "test-cluster-token",
  "security": {
    "blacklisted_commands": ["rm", "mkfs", "shutdown", "reboot"],
    "dangerous_args_regex": []
  }
}
EOF

# Node 2 config
cat > test_node2_config.json << EOF
{
  "port": 8081,
  "node_name": "node-02",
  "peers": ["http://localhost:8080", "http://localhost:8082"],
  "cluster_token": "test-cluster-token",
  "security": {
    "blacklisted_commands": ["rm", "mkfs", "shutdown", "reboot"],
    "dangerous_args_regex": []
  }
}
EOF

# Node 3 config
cat > test_node3_config.json << EOF
{
  "port": 8082,
  "node_name": "node-03",
  "peers": ["http://localhost:8080", "http://localhost:8081"],
  "cluster_token": "test-cluster-token",
  "security": {
    "blacklisted_commands": ["rm", "mkfs", "shutdown", "reboot"],
    "dangerous_args_regex": []
  }
}
EOF

# Client config (connects to node-01)
cat > test_cluster_client_config.json << EOF
{
  "servers": [
    {
      "name": "node-01",
      "url": "http://localhost:8080"
    },
    {
      "name": "node-02",
      "url": "http://localhost:8081"
    }
  ]
}
EOF

print_success "Configuration files created"

# Start all 3 servers in the background
print_info "Starting 3 server nodes..."

# Start Node 1
if [ -f "cmd/server/server.exe" ]; then
    cmd/server/server.exe test_node1_config.json > node1.log 2>&1 &
elif [ -f "cmd/server/server" ]; then
    cmd/server/server test_node1_config.json > node1.log 2>&1 &
fi
NODE1_PID=$!
print_success "Node 1 started on port 8080 (PID: $NODE1_PID)"

# Start Node 2
if [ -f "cmd/server/server.exe" ]; then
    cmd/server/server.exe test_node2_config.json > node2.log 2>&1 &
elif [ -f "cmd/server/server" ]; then
    cmd/server/server test_node2_config.json > node2.log 2>&1 &
fi
NODE2_PID=$!
print_success "Node 2 started on port 8081 (PID: $NODE2_PID)"

# Start Node 3
if [ -f "cmd/server/server.exe" ]; then
    cmd/server/server.exe test_node3_config.json > node3.log 2>&1 &
elif [ -f "cmd/server/server" ]; then
    cmd/server/server test_node3_config.json > node3.log 2>&1 &
fi
NODE3_PID=$!
print_success "Node 3 started on port 8082 (PID: $NODE3_PID)"

# Wait for servers to be ready
print_info "Waiting for all servers to be ready..."
sleep 5

# Check if all servers are running
if ! kill -0 $NODE1_PID 2>/dev/null || ! kill -0 $NODE2_PID 2>/dev/null || ! kill -0 $NODE3_PID 2>/dev/null; then
    print_error "One or more servers failed to start. Check logs for details."
    echo "Node 1 log:"
    cat node1.log
    echo "Node 2 log:"
    cat node2.log
    echo "Node 3 log:"
    cat node3.log
    exit 1
fi

print_success "All 3 servers are running"

# Run tests
echo ""
echo "=========================================="
echo "Running Cluster Tests"
echo "=========================================="
echo ""

# Test 1: Echo command on all nodes
print_info "Test 1: Running 'echo Hello Cluster' on all nodes..."
echo "echo Hello Cluster" | timeout 10 cmd/client/client.exe test_cluster_client_config.json > cluster_output.log 2>&1 || true

if grep -q "Hello Cluster" cluster_output.log; then
    print_success "Test 1 PASSED: Command executed on cluster"
    # Check if results from multiple nodes are aggregated
    if grep -q "3 nodes\|3 groups\|Count: 3" cluster_output.log; then
        print_success "  Results from all 3 nodes were aggregated"
    else
        print_info "  Output may show individual node results"
    fi
else
    print_error "Test 1 FAILED: Command did not produce expected output"
    cat cluster_output.log
fi

# Test 2: Hostname command (should return different hostnames)
print_info "Test 2: Running 'hostname' on all nodes..."
echo "hostname" | timeout 10 cmd/client/client.exe test_cluster_client_config.json > cluster_output.log 2>&1 || true

if grep -q "node-01\|node-02\|node-03" cluster_output.log; then
    print_success "Test 2 PASSED: Hostname command executed on cluster"
    if grep -q "node-01" cluster_output.log && grep -q "node-02" cluster_output.log && grep -q "node-03" cluster_output.log; then
        print_success "  Results from all 3 nodes present"
    fi
else
    print_error "Test 2 FAILED: Hostname command did not work"
    cat cluster_output.log
fi

# Test 3: Security check on cluster
print_info "Test 3: Running 'rm -rf /' (should be blocked on all nodes)..."
echo "rm -rf /" | timeout 10 cmd/client/client.exe test_cluster_client_config.json > cluster_output.log 2>&1 || true

if grep -q "security violation\|blacklisted" cluster_output.log; then
    print_success "Test 3 PASSED: Security guard blocked dangerous command on cluster"
else
    print_error "Test 3 FAILED: Security guard did not block the command"
    cat cluster_output.log
fi

# Test 4: Date command (should show execution on all nodes)
print_info "Test 4: Running 'date' on all nodes..."
echo "date" | timeout 10 cmd/client/client.exe test_cluster_client_config.json > cluster_output.log 2>&1 || true

if grep -q "202[0-9]" cluster_output.log; then
    print_success "Test 4 PASSED: Date command executed on cluster"
else
    print_error "Test 4 FAILED: Date command did not work"
    cat cluster_output.log
fi

# Cleanup
echo ""
echo "=========================================="
echo "Cleanup"
echo "=========================================="

print_info "Stopping all servers..."
kill $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null || true
wait $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null || true
print_success "All servers stopped"

# Clean up test files
rm -f test_node1_config.json test_node2_config.json test_node3_config.json
rm -f test_cluster_client_config.json
rm -f node1.log node2.log node3.log cluster_output.log

echo ""
echo "=========================================="
echo "Cluster Test Complete"
echo "=========================================="
