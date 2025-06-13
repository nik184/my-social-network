#!/bin/bash

set -e

echo "🚀 Starting E2E tests for P2P Social Network"

# Cleanup any existing containers
echo "🧹 Cleaning up existing containers..."
docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing npm dependencies..."
    npm install
fi

# Install Playwright browsers if needed
echo "🎭 Ensuring Playwright browsers are installed..."
npx playwright install chromium

# Start containers
echo "🐳 Starting Docker containers..."
docker-compose -f docker-compose.test.yml up -d --build

# Wait for containers to be healthy
echo "⏳ Waiting for containers to be healthy..."
timeout=120
counter=0

while [ $counter -lt $timeout ]; do
    node1_health=$(docker inspect --format='{{.State.Health.Status}}' social-network-node1 2>/dev/null || echo "starting")
    node2_health=$(docker inspect --format='{{.State.Health.Status}}' social-network-node2 2>/dev/null || echo "starting")
    
    if [ "$node1_health" = "healthy" ] && [ "$node2_health" = "healthy" ]; then
        echo "✅ Both containers are healthy!"
        break
    fi
    
    echo "Waiting... Node1: $node1_health, Node2: $node2_health (${counter}s/${timeout}s)"
    sleep 2
    counter=$((counter + 2))
done

if [ $counter -ge $timeout ]; then
    echo "❌ Containers failed to become healthy within $timeout seconds"
    docker-compose -f docker-compose.test.yml logs
    docker-compose -f docker-compose.test.yml down -v
    exit 1
fi

# Additional wait for full application startup
echo "⏳ Waiting for applications to fully start..."
sleep 10

# Run the tests
echo "🧪 Running E2E tests..."
if npx playwright test; then
    echo "✅ E2E tests passed!"
    test_result=0
else
    echo "❌ E2E tests failed!"
    test_result=1
fi

# Cleanup
echo "🧹 Cleaning up containers..."
docker-compose -f docker-compose.test.yml down -v

if [ $test_result -eq 0 ]; then
    echo "🎉 All tests completed successfully!"
else
    echo "💥 Tests failed. Check the logs above for details."
fi

exit $test_result