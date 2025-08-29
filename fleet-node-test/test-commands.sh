#!/bin/bash

# Fleet Node.js Test Script
# This script tests various aspects of the Node.js runtime support

echo "================================"
echo "Fleet Node.js Runtime Test Suite"
echo "================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test function
test_command() {
    local description="$1"
    local command="$2"
    
    echo -e "${YELLOW}Testing:${NC} $description"
    echo "Command: $command"
    
    if eval "$command"; then
        echo -e "${GREEN}✓ Test passed${NC}"
    else
        echo -e "${RED}✗ Test failed${NC}"
    fi
    echo ""
}

# Change to fleet-node-test directory
cd /home/argent/fleet/fleet-node-test

echo "1. Starting Fleet services..."
echo "------------------------------"
test_command "Fleet up" "../fleet up -d"

# Wait for services to start
echo "Waiting for services to start..."
sleep 10

echo "2. Testing fleet-node CLI commands..."
echo "--------------------------------------"

# Test npm commands
test_command "List npm packages in Express API" "../fleet-node --service=api npm list --depth=0"

test_command "Run npm test in Express API" "../fleet-node --service=api npm test"

# Test with different services
test_command "Check Node version in Next.js" "../fleet-node --service=nextjs node --version"

test_command "Check npm version in Vue app" "../fleet-node --service=vue npm --version"

# Test package installation (without actually installing)
test_command "Check if we can run npm install (dry-run)" "../fleet-node --service=api npm install --dry-run"

echo "3. Testing service endpoints..."
echo "--------------------------------"

# Test Express API
test_command "Express API health check" "curl -s http://localhost:3000/health | jq ."

test_command "Express API users endpoint" "curl -s http://localhost:3000/api/users | jq ."

# Test Next.js
test_command "Next.js home page" "curl -s -o /dev/null -w '%{http_code}' http://localhost:3001"

test_command "Next.js API route" "curl -s http://localhost:3001/api/hello | jq ."

# Test Vue (if running)
test_command "Vue.js app" "curl -s -o /dev/null -w '%{http_code}' http://localhost:8080"

echo "4. Testing Docker containers..."
echo "--------------------------------"

test_command "List all containers" "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"

test_command "Check Node containers" "docker ps --filter 'name=node' --format 'table {{.Names}}\t{{.Image}}'"

echo "5. Testing build mode (React)..."
echo "---------------------------------"

# Check if build artifacts exist
test_command "Check React build directory" "ls -la react-frontend/build 2>/dev/null || echo 'Build directory not created yet'"

# Check nginx is serving React
test_command "React app via nginx" "curl -s -o /dev/null -w '%{http_code}' http://localhost:80"

echo "6. Fleet status and logs..."
echo "----------------------------"

test_command "Fleet status" "../fleet status"

echo "Sample logs from Express API:"
../fleet logs api --tail=5

echo ""
echo "================================"
echo "Test Summary"
echo "================================"
echo ""
echo "Test Environment Details:"
echo "- Project: node-test"
echo "- Services: api (Express), nextjs, react (nginx), vue"
echo "- Node versions: 20 (api, nextjs, react), 18 (vue)"
echo "- Modes tested: Service mode (api, nextjs, vue), Build mode (react)"
echo ""
echo "To clean up:"
echo "  ../fleet down"
echo ""
echo "To connect to a container:"
echo "  ../fleet-node --service=api /bin/sh"
echo ""
echo "To run framework commands:"
echo "  ../fleet-node --service=nextjs npm run dev"
echo "  ../fleet-node --service=vue npm run serve"