# E2E Testing for P2P Social Network

This directory contains End-to-End (E2E) tests for the P2P Social Network application using Docker containers and Playwright.

## Overview

The E2E tests validate the core P2P functionality by:
1. Starting two containerized instances of the application
2. Testing basic functionality on both nodes
3. Attempting to connect one node to another as a friend
4. Verifying the connection works
5. Cleaning up containers after tests

## Files Structure

```
├── Dockerfile                 # Container definition for the app
├── docker-compose.test.yml    # Two-node test environment
├── package.json              # Node.js dependencies for testing
├── playwright.config.js      # Playwright configuration
├── run-e2e-tests.sh         # Test runner script
├── e2e/
│   ├── basic-functionality.spec.js  # Basic app functionality tests
│   └── friend-connection.spec.js    # P2P friend connection tests
└── E2E_TESTING.md           # This file
```

## Prerequisites

- Docker and Docker Compose
- Node.js (for Playwright)
- bash (for the test runner script)

## Running the Tests

### Option 1: Using the Test Runner Script (Recommended)

```bash
# Make the script executable (if not already)
chmod +x run-e2e-tests.sh

# Run all E2E tests
./run-e2e-tests.sh
```

### Option 2: Manual Steps

```bash
# Install dependencies
npm install

# Install Playwright browsers
npx playwright install chromium

# Start the containers
docker-compose -f docker-compose.test.yml up -d --build

# Wait for containers to be healthy (check with docker ps)
# Then run tests
npx playwright test

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

### Option 3: Using npm scripts

```bash
# Install dependencies first
npm install

# Run the full test suite (starts containers, runs tests, cleans up)
npm run test:e2e

# Or run individual steps
npm run docker:setup    # Start containers
npm run test           # Run tests only
npm run docker:cleanup # Stop containers
```

## Test Environment

The Docker Compose setup creates:

- **Node 1**: 
  - Web interface: http://localhost:6996
  - P2P port: 9000
  - Container name: social-network-node1

- **Node 2**: 
  - Web interface: http://localhost:6997
  - P2P port: 9001
  - Container name: social-network-node2

Both nodes run in the same Docker network and can communicate with each other.

## Test Scenarios

### Basic Functionality Tests (`basic-functionality.spec.js`)
- ✅ Node 1 loads homepage and redirects to profile
- ✅ Node 2 loads homepage and redirects to profile  
- ✅ Both nodes can navigate to friends page
- ✅ API endpoints respond with valid data
- ✅ Nodes have different peer IDs

### Friend Connection Tests (`friend-connection.spec.js`)
- ✅ Get node information from both instances
- ✅ Create connection string for Node 1
- ✅ Add Node 1 as friend from Node 2
- ✅ Verify friend appears in Node 2's friends list
- ✅ Test basic P2P connectivity

## Debugging Failed Tests

### Check Container Logs
```bash
# View logs from both containers
docker-compose -f docker-compose.test.yml logs

# View logs from specific container
docker logs social-network-node1
docker logs social-network-node2
```

### Check Container Health
```bash
# Check if containers are running and healthy
docker ps

# Inspect health status
docker inspect --format='{{.State.Health.Status}}' social-network-node1
docker inspect --format='{{.State.Health.Status}}' social-network-node2
```

### Manual Testing
```bash
# Start containers
docker-compose -f docker-compose.test.yml up -d --build

# Test manually in browser
# Node 1: http://localhost:6996
# Node 2: http://localhost:6997

# Check API endpoints
curl http://localhost:6996/api/info
curl http://localhost:6997/api/info
```

### Playwright Debug Mode
```bash
# Run tests in headed mode (shows browser)
npx playwright test --headed

# Run with debug mode
npx playwright test --debug

# Run specific test file
npx playwright test e2e/basic-functionality.spec.js
```

## Troubleshooting

### Container Startup Issues
- Ensure Docker is running
- Check for port conflicts (6996, 6997, 9000, 9001)
- Verify Go build completes successfully
- Check application logs for startup errors

### P2P Connection Issues
- Verify both nodes are healthy before testing
- Check that containers can reach each other in Docker network
- Ensure P2P ports are properly exposed
- Verify connection string format is correct

### Test Timeout Issues
- Increase timeouts in playwright.config.js
- Add more wait time in test setup
- Check if containers need more time to fully start

## Extending the Tests

To add more test scenarios:

1. Create new `.spec.js` files in the `e2e/` directory
2. Follow the existing test structure
3. Use the established beforeAll/afterAll setup for Docker containers
4. Test different aspects of P2P functionality (file sharing, messaging, etc.)

## Performance Considerations

- Tests run sequentially (not in parallel) to avoid P2P conflicts
- Container startup can take 30-60 seconds
- Full test suite typically takes 2-3 minutes
- Consider running on machines with adequate resources (2GB+ RAM recommended)