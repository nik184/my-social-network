# Integration Tests

This directory contains comprehensive integration tests for the distributed social network application.

## Test Categories

### 1. Direct Connection Test (`direct_connection_test.go`)
Tests basic peer-to-peer connectivity:
- Two nodes connect directly
- Names are exchanged during identification
- Connection information is stored in databases
- Peer validation works properly

**What it tests:**
- Basic P2P connectivity
- Peer identification protocol
- Name exchange mechanism
- Database persistence
- Connection validation

### 2. Hole Punching Test (`hole_punching_test.go`)
Tests advanced hole punching functionality:
- Three nodes (A, B, Relay) are created
- A and B connect to Relay but not directly to each other
- A discovers B as second-degree connection through Relay
- A initiates hole punching to connect directly to B
- Direct connection is established and data is exchanged

**What it tests:**
- Second-degree peer discovery
- Hole punching protocol
- Relay-assisted connections
- NAT traversal simulation
- Multi-hop network topology

### 3. Containerized Tests (`containerized_test.go`)
Tests the application in containerized environments:
- Uses testcontainers-go for realistic network simulation
- Tests container-to-container communication
- Simulates NAT and firewall conditions
- Tests network partitioning and recovery

**What it tests:**
- Real network conditions
- Container networking
- Network isolation
- Latency and packet loss handling

## Test Infrastructure

### Test Helpers (`test_helpers.go`)
Provides utility functions and structures:
- `TestNode`: Wrapper for application nodes in tests
- `TestNetwork`: Manages collections of test nodes
- Network topology creation (star, mesh)
- Connection helpers and verification

## Running Tests

### Prerequisites
- Docker (for containerized tests)
- Go 1.23+
- Sufficient disk space for temporary test directories

### Run All Tests
```bash
go test ./tests/integration/...
```

### Run Specific Tests
```bash
# Direct connection test only
go test ./tests/integration/ -run TestDirectConnection

# Hole punching test only  
go test ./tests/integration/ -run TestHolePunching

# Skip containerized tests (faster)
go test ./tests/integration/ -short
```

### Verbose Output
```bash
go test ./tests/integration/ -v
```

## Test Scenarios Covered

### Basic Connectivity
- ✅ Two nodes connect directly
- ✅ Names are exchanged
- ✅ Connection info is stored
- ✅ Peer validation works

### Advanced Networking
- ✅ Second-degree peer discovery
- ✅ Hole punching through relay
- ✅ Multi-relay scenarios
- ✅ Complex network topologies

### Edge Cases
- ✅ Empty peer lists (EOF handling)
- ✅ Connection failures
- ✅ Invalid peer IDs
- ✅ Network timeouts

### Real-World Conditions
- 🔄 Containerized environments
- 🔄 Network partitions
- 🔄 High latency networks
- 🔄 Packet loss scenarios

## Test Architecture

The tests are designed to:

1. **Isolate Components**: Each test creates independent node instances
2. **Simulate Real Conditions**: Use temporary directories and realistic network setups
3. **Verify End-to-End Behavior**: Test complete workflows from connection to data exchange
4. **Handle Cleanup**: Automatic cleanup of resources and temporary files
5. **Provide Clear Feedback**: Detailed logging and assertions

## Debugging Tests

### Common Issues
1. **Port Conflicts**: Tests use dynamic port allocation to avoid conflicts
2. **Timing Issues**: Tests include stabilization periods for network operations
3. **Resource Cleanup**: Automatic cleanup prevents resource leaks

### Debug Logging
Tests include detailed logging to help diagnose issues:
- Node startup and configuration
- Connection attempts and results
- Peer discovery progress
- Database operations

### Manual Verification
You can manually verify test results by:
1. Checking temporary directories during test execution
2. Examining database contents
3. Monitoring network connections
4. Reviewing log output

## Contributing

When adding new tests:
1. Use the existing test helpers where possible
2. Include proper cleanup in defer statements
3. Add descriptive test names and comments
4. Verify tests work in isolation and in parallel
5. Update this README with new test scenarios