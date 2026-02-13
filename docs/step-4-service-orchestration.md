# Step 4: Service Orchestration Implementation

## Overview
Implemented service orchestration with full dry-run support as specified in step 4 of the development plan.

## Components Created

### 1. Service Package (`internal/services/services.go`)
Core service orchestration functionality:

- **Process struct**: Represents a running service with name, config, command, and PID
- **StartService()**: Starts a service with dry-run support
  - In dry-run mode: logs what would be done without executing
  - In real mode: spawns process and tracks PID
  - Redirects stdout/stderr to /dev/null
  - Starts processes in their own process group (Setpgid)
- **Process.Stop()**: Gracefully stops a service
  - Sends SIGTERM for graceful shutdown
  - Waits for process to exit
  - Handles already-exited processes gracefully
- **Process.IsRunning()**: Checks if process is still alive
  - Uses signal(0) to check process existence
- **StopAllServices()**: Stops all services in reverse order (LIFO)

### 2. Logger Package (`internal/logger/logger.go`)
Colored structured logging matching bash script style:

- Log levels: Info, Success, Warning, Error, Verbose
- Color-coded output using fatih/color:
  - Info: Cyan
  - Success: Green
  - Warning: Yellow
  - Error: Red
  - Verbose: Faint white
- Format: `[LEVEL] message`
- Verbose logging can be enabled/disabled with `SetVerbose()`

### 3. Tests (`internal/services/services_test.go`)
Comprehensive test coverage:

- `TestStartService_DryRun`: Verifies dry-run mode behavior
- `TestStartService_Real`: Tests actual service startup
- `TestStopService_DryRun`: Tests dry-run stop behavior
- `TestStopAllServices`: Tests stopping multiple services
- `TestJoinArgs`: Tests command argument formatting
- `TestIsRunning`: Tests process state detection

### 4. Example (`examples/service_example.go`)
Demonstrates service orchestration:

- Starts multiple services (database, api-server, worker)
- Supports `--dry-run` flag
- Implements signal handling (SIGINT/SIGTERM)
- Shows graceful shutdown
- Can be run with: `go run examples/service_example.go [--dry-run]`

## Key Features

### Dry-Run Support
- All service operations respect the dry-run flag
- In dry-run mode:
  - Logs planned actions with "Would start service: ..."
  - Returns Process with PID=-1 (sentinel value)
  - No actual processes spawned
  - StopAllServices() logs "Would stop service: ..."

### Process Management
- Services started in separate process groups
- Graceful shutdown with SIGTERM
- Proper cleanup on exit
- Process state tracking

### Integration with Config
- Uses `config.Service` struct from existing config package
- Reads service definitions from .ralph/config.yaml
- Supports optional port field for health checking (to be implemented in step 5)

## Dependencies Added
- `github.com/fatih/color` v1.18.0 - for colored terminal output

## Testing
All tests pass successfully:
```
go test ./... -v
```

Example output:
- Dry-run mode: Shows planned actions without execution
- Real mode: Starts processes, runs for 3 seconds, then stops gracefully

## Next Steps
Step 5 will implement health checking for services:
- `WaitForPort()`: Wait for TCP port to be available
- `CheckPort()`: Check if TCP port is listening
- Handle services without ports (verify process running only)

## Files Created/Modified
- ✅ `internal/services/services.go` - Core service orchestration
- ✅ `internal/services/services_test.go` - Comprehensive tests
- ✅ `internal/logger/logger.go` - Structured logging
- ✅ `examples/service_example.go` - Working example
- ✅ `go.mod` - Added fatih/color dependency
- ✅ `docs/step-4-service-orchestration.md` - This documentation
