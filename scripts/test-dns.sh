#!/bin/bash

# Fleet DNS Test Script
# Tests the dnsmasq configuration and .test domain resolution

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Fleet DNS Test Script${NC}"
echo "===================="

# Check if dnsmasq container is running
echo -e "\n${YELLOW}Checking dnsmasq container status...${NC}"
if docker ps | grep -q fleet-dnsmasq; then
    echo -e "${GREEN}✓ Dnsmasq container is running${NC}"
else
    echo -e "${RED}✗ Dnsmasq container is not running${NC}"
    echo "Start it with: docker-compose -f templates/compose/docker-compose.dnsmasq.yml up -d"
    exit 1
fi

# Test DNS resolution using different methods
echo -e "\n${YELLOW}Testing DNS resolution for .test domains...${NC}"

# Test domains
TEST_DOMAINS=("test.test" "app.test" "api.test" "dnsmasq.test")

for domain in "${TEST_DOMAINS[@]}"; do
    echo -e "\nTesting ${YELLOW}$domain${NC}:"
    
    # Using nslookup
    if command -v nslookup &> /dev/null; then
        result=$(nslookup "$domain" 127.0.0.1 2>&1 | grep -A1 "Name:" | tail -1 || echo "Failed")
        if echo "$result" | grep -q "127.0.0.1\|172.20"; then
            echo -e "  nslookup: ${GREEN}✓ Resolved${NC}"
        else
            echo -e "  nslookup: ${RED}✗ Failed${NC}"
        fi
    fi
    
    # Using dig
    if command -v dig &> /dev/null; then
        result=$(dig "@127.0.0.1" "$domain" +short 2>&1)
        if [ ! -z "$result" ] && [[ "$result" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            echo -e "  dig:      ${GREEN}✓ Resolved to $result${NC}"
        else
            echo -e "  dig:      ${RED}✗ Failed${NC}"
        fi
    fi
    
    # Using host
    if command -v host &> /dev/null; then
        result=$(host "$domain" 127.0.0.1 2>&1)
        if echo "$result" | grep -q "has address"; then
            echo -e "  host:     ${GREEN}✓ Resolved${NC}"
        else
            echo -e "  host:     ${RED}✗ Failed${NC}"
        fi
    fi
done

# Check container logs for recent queries
echo -e "\n${YELLOW}Recent DNS queries (last 10 lines):${NC}"
docker logs fleet-dnsmasq --tail 10 2>&1 | grep -E "query|reply" || echo "No recent queries found"

echo -e "\n${GREEN}Test complete!${NC}"
echo -e "\nTo add custom .test domains:"
echo -e "  1. Edit: config/services/hosts.test"
echo -e "  2. Restart: docker-compose -f templates/compose/docker-compose.dnsmasq.yml restart"