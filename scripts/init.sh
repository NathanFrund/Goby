#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   Goby Framework Initialization${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
    fi
}

# Check for required tools
echo -e "${BLUE}Checking prerequisites...${NC}"
echo ""

# Check Go
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_status 0 "Go is installed ($GO_VERSION)"
else
    print_status 1 "Go is not installed"
    echo -e "${YELLOW}  Install from: https://golang.org/dl/${NC}"
    exit 1
fi

# Check Node.js
if command_exists node; then
    NODE_VERSION=$(node --version)
    print_status 0 "Node.js is installed ($NODE_VERSION)"
else
    print_status 1 "Node.js is not installed"
    echo -e "${YELLOW}  Install from: https://nodejs.org/${NC}"
    exit 1
fi

# Check npm
if command_exists npm; then
    NPM_VERSION=$(npm --version)
    print_status 0 "npm is installed (v$NPM_VERSION)"
else
    print_status 1 "npm is not installed"
    exit 1
fi

# Check Overmind (optional but recommended)
if command_exists overmind; then
    print_status 0 "Overmind is installed"
else
    print_status 1 "Overmind is not installed (optional)"
    echo -e "${YELLOW}  Install with: go install github.com/DarthSim/overmind/v2@latest${NC}"
    echo -e "${YELLOW}  Or on macOS: brew install overmind${NC}"
    echo ""
fi

echo ""
echo -e "${BLUE}Installing dependencies...${NC}"
echo ""

# Install Node dependencies
echo "Installing Node.js dependencies..."
if npm install; then
    print_status 0 "Node.js dependencies installed"
else
    print_status 1 "Failed to install Node.js dependencies"
    exit 1
fi

# Install Go dependencies
echo "Installing Go dependencies..."
if go mod download; then
    print_status 0 "Go dependencies installed"
else
    print_status 1 "Failed to install Go dependencies"
    exit 1
fi

echo ""
echo -e "${BLUE}Setting up configuration...${NC}"
echo ""

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "Creating .env file from .env.minimal..."
    if [ -f .env.minimal ]; then
        cp .env.minimal .env
        print_status 0 ".env file created from .env.minimal"
    elif [ -f .env.example ]; then
        cp .env.example .env
        print_status 0 ".env file created from .env.example"
    else
        print_status 1 "No .env template found"
    fi
    
    echo -e "${YELLOW}  ⚠ Remember to update .env with your actual configuration!${NC}"
else
    print_status 0 ".env file already exists"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}   Initialization Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo ""
echo "1. Start SurrealDB (choose one method):"
echo ""
echo -e "   ${BLUE}Option A - Docker:${NC}"
echo -e "   ${BLUE}docker run --rm -p 8000:8000 surrealdb/surrealdb:latest \\${NC}"
echo -e "   ${BLUE}     start --log trace --user root --pass root${NC}"
echo ""
echo -e "   ${BLUE}Option B - Native binary:${NC}"
echo -e "   ${BLUE}surreal start --log trace --user root --pass root${NC}"
echo ""
echo -e "   ${BLUE}Option C - Existing instance:${NC}"
echo -e "   ${BLUE}Update .env with your SurrealDB connection details${NC}"
echo ""
echo "2. Update your .env file with database credentials (if needed)"
echo ""
echo "3. Start the development server:"
echo -e "   ${BLUE}make dev${NC}"
echo ""
echo "4. Open your browser to:"
echo -e "   ${BLUE}http://localhost:8080${NC}"
echo ""
echo -e "${YELLOW}Helpful commands:${NC}"
echo -e "  ${BLUE}make help${NC}                              - Show all available commands"
echo -e "  ${BLUE}go run ./cmd/goby-cli new-module --name=myfeature${NC} - Create a new module"
echo -e "  ${BLUE}go run ./cmd/goby-cli list-services${NC}    - List all services"
echo ""
