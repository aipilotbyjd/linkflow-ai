#!/bin/bash

# LinkFlow AI - Test Services Script
# This script demonstrates API calls to test the microservices

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "🧪 LinkFlow AI - Service Testing"
echo "================================"
echo ""

# Base URLs
AUTH_URL="http://localhost:8001"
USER_URL="http://localhost:8002"
WORKFLOW_URL="http://localhost:8004"

# Test data
TEST_EMAIL="test@example.com"
TEST_USERNAME="testuser"
TEST_PASSWORD="Test123!Pass"

echo "📋 Testing Service Health Checks"
echo "---------------------------------"

# Test Auth Service
echo -e "${BLUE}Testing Auth Service...${NC}"
if curl -s -f "$AUTH_URL/health/ready" > /dev/null; then
    echo -e "${GREEN}✓ Auth Service is healthy${NC}"
else
    echo -e "${RED}✗ Auth Service is not responding${NC}"
fi

# Test User Service
echo -e "${BLUE}Testing User Service...${NC}"
if curl -s -f "$USER_URL/health/ready" > /dev/null; then
    echo -e "${GREEN}✓ User Service is healthy${NC}"
else
    echo -e "${RED}✗ User Service is not responding${NC}"
fi

# Test Workflow Service
echo -e "${BLUE}Testing Workflow Service...${NC}"
if curl -s -f "$WORKFLOW_URL/health/ready" > /dev/null; then
    echo -e "${GREEN}✓ Workflow Service is healthy${NC}"
else
    echo -e "${RED}✗ Workflow Service is not responding${NC}"
fi

echo ""
echo "🔐 Testing User Registration & Login"
echo "------------------------------------"

# Register a new user
echo -e "${BLUE}Registering new user...${NC}"
REGISTER_RESPONSE=$(curl -s -X POST "$USER_URL/api/v1/users/register" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "'$TEST_EMAIL'",
        "username": "'$TEST_USERNAME'",
        "password": "'$TEST_PASSWORD'",
        "firstName": "Test",
        "lastName": "User"
    }')

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ User registered successfully${NC}"
    echo "Response: $REGISTER_RESPONSE" | jq '.' 2>/dev/null || echo "$REGISTER_RESPONSE"
else
    echo -e "${RED}✗ User registration failed${NC}"
fi

echo ""

# Login
echo -e "${BLUE}Logging in...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$USER_URL/api/v1/users/login" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "'$TEST_EMAIL'",
        "password": "'$TEST_PASSWORD'"
    }')

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Login successful${NC}"
    # Extract token if available
    TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token' 2>/dev/null)
    if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
        echo "Token received: ${TOKEN:0:20}..."
    fi
else
    echo -e "${RED}✗ Login failed${NC}"
fi

echo ""
echo "📝 Testing Workflow Operations"
echo "------------------------------"

# Create a workflow
echo -e "${BLUE}Creating a workflow...${NC}"
WORKFLOW_RESPONSE=$(curl -s -X POST "$WORKFLOW_URL/api/v1/workflows" \
    -H "Content-Type: application/json" \
    -H "X-User-ID: test-user-123" \
    -d '{
        "name": "My Test Workflow",
        "description": "A test workflow for demonstration",
        "nodes": [
            {
                "id": "node1",
                "type": "trigger",
                "name": "HTTP Trigger",
                "config": {},
                "position": {"x": 100, "y": 100}
            },
            {
                "id": "node2",
                "type": "action",
                "name": "Send Email",
                "config": {"to": "user@example.com"},
                "position": {"x": 300, "y": 100}
            }
        ],
        "connections": [
            {
                "id": "conn1",
                "sourceNodeId": "node1",
                "targetNodeId": "node2"
            }
        ]
    }')

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Workflow created successfully${NC}"
    WORKFLOW_ID=$(echo "$WORKFLOW_RESPONSE" | jq -r '.id' 2>/dev/null)
    echo "Workflow ID: $WORKFLOW_ID"
else
    echo -e "${RED}✗ Workflow creation failed${NC}"
fi

# List workflows
echo ""
echo -e "${BLUE}Listing workflows...${NC}"
LIST_RESPONSE=$(curl -s -X GET "$WORKFLOW_URL/api/v1/workflows" \
    -H "X-User-ID: test-user-123")

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Workflows retrieved successfully${NC}"
    WORKFLOW_COUNT=$(echo "$LIST_RESPONSE" | jq '.total' 2>/dev/null)
    echo "Total workflows: $WORKFLOW_COUNT"
else
    echo -e "${RED}✗ Failed to list workflows${NC}"
fi

echo ""
echo "📊 Summary"
echo "----------"
echo ""
echo "Available API Endpoints:"
echo ""
echo "Auth Service (Port 8001):"
echo "  POST /auth/login"
echo "  POST /auth/register"
echo "  POST /auth/refresh"
echo "  POST /auth/logout"
echo ""
echo "User Service (Port 8002):"
echo "  POST /api/v1/users/register"
echo "  POST /api/v1/users/login"
echo "  GET  /api/v1/users/profile"
echo "  PUT  /api/v1/users/profile"
echo "  GET  /api/v1/users/{id}"
echo "  POST /api/v1/users/change-password"
echo "  POST /api/v1/organizations"
echo ""
echo "Workflow Service (Port 8004):"
echo "  POST /api/v1/workflows"
echo "  GET  /api/v1/workflows"
echo "  GET  /api/v1/workflows/{id}"
echo "  PUT  /api/v1/workflows/{id}"
echo "  DELETE /api/v1/workflows/{id}"
echo "  POST /api/v1/workflows/{id}/activate"
echo "  POST /api/v1/workflows/{id}/deactivate"
echo "  POST /api/v1/workflows/{id}/duplicate"
echo ""
echo -e "${GREEN}✨ Testing complete!${NC}"
