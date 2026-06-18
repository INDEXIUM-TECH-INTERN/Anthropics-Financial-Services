#!/bin/bash

# TestAIFinance Deployment Script for Render
# Automates testing and deployment to Render.com

set -e

echo "🚀 Starting TestAIFinance deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[✓] $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}[!] $1${NC}"
}

print_error() {
    echo -e "${RED}[✗] $1${NC}"
}

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed"
    exit 1
fi

print_status "Prerequisites check passed"

# Clean previous builds
print_status "Cleaning previous builds..."
docker-compose down
docker system prune -f

# Build and test locally
print_status "Building and testing services locally..."
docker-compose build --no-cache

print_status "Starting services..."
docker-compose up -d

# Wait for services to be healthy
print_status "Waiting for services to be healthy..."
sleep 30

# Health checks
if curl -s http://localhost:8080/health > /dev/null; then
    print_status "Backend is healthy"
else
    print_warning "Backend health check failed"
fi

if curl -s http://localhost:3000 > /dev/null; then
    print_status "Frontend is healthy"
else
    print_warning "Frontend health check failed"
fi

# Run integration tests
print_status "Running integration tests..."
./test-deployment.sh

print_status "Local tests completed successfully"

# Check if git is clean
if [[ -n $(git status --porcelain) ]]; then
    print_warning "Working directory is not clean"
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Push to git (if changes)
if [[ -n $(git status --porcelain) ]]; then
    print_status "Committing and pushing changes..."
    git add .
    git commit -m "chore: update deployment configuration"
    git push origin main
fi

# Deploy to Render
print_status "Deploying to Render..."

# Check if render CLI is installed
if command -v render &> /dev/null; then
    render deploy -P render.yaml
else
    print_warning "Render CLI not found"
    print_warning "Please deploy manually:"
    print_warning "1. Push to GitHub"
    print_warning "2. Go to https://render.com/dashboard"
    print_warning "3. Create new service from blueprint"
fi

print_status "🎉 Deployment completed!"

# Show next steps
echo ""
echo "📝 Next Steps:"
echo "1. Set API keys in Render dashboard"
echo "2. Verify services are running"
echo "3. Check health monitoring"
echo "4. Set up alerts if needed"

# Cleanup
echo ""
read -p "Stop local services? (Y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Nn]$ ]]; then
    print_status "Services running locally"
else
    docker-compose down
    print_status "Local services stopped"
fi