# Design Document

## Overview

This design has been successfully implemented to add debug message logging for the bubbletea-based interactive CLI application following the approach described in the leg100 blog post. The solution adds a debug writer field to the Model struct and uses the `spew` package to pretty-print all bubbletea messages to a log file when debug mode is enabled.

The implementation is activated by a `--debug` flag and logs all messages to a file in the `~/.zamm/logs` directory using spew for readable formatting.

## Architecture

### High-Level Components

1. **Debug Writer Integration**: Add `io.Writer` field to the Model struct for message dumping
2. **Debug Flag Handler**: Command-line flag processing to enable/disable debug mode  
3. **Log File Manager**: Creates and manages debug log files in `~/.zamm/logs`
4. **Message Dumping**: Use `spew.Fdump()` in the Update method to log all messages

### Integration Points

- **CLI Root Command**: Add `--debug` flag to enable debug logging
- **Model Struct**: Add debug writer field to capture messages
- **Update Method**: Add message dumping logic to log all received messages
- **Interactive Mode**: Pass debug writer to model when debug flag is set

## Components and Interfaces

### 1. Enhanced Model Structure

```go
type Model struct {
    // ... existing fields ...
    debugWriter io.Writer // Added for debug message dumping
}
```

### 2. Debug Log File Creation

```go
func createDebugLogFile(logDir string) (*os.File, error)
```

### 3. Debug Integration in Update Method

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.debugWriter != nil {
        spew.Fdump(m.debugWriter, msg)
    }
    // ... existing update logic ...
}
```

## Data Models

### Message Logging Format

The `spew` package will handle all message formatting and type detection automatically. Messages will be dumped in a human-readable format showing:
- Message type (e.g., `(tea.KeyMsg)`, `(tea.WindowSizeMsg)`)
- Message contents with field names and values
- Nested structures properly indented

Example output:
```
(tea.WindowSizeMsg) {
 Width: (int) 127,
 Height: (int) 30
}
(tea.KeyMsg) {
 Type: (tea.KeyType) 1,
 Runes: ([]rune) <nil>,
 Alt: (bool) false
}
```

## Error Handling

### Log File Creation Errors
- If log directory cannot be created, fall back to current directory
- If log file cannot be created, disable debug logging and continue normal operation
- Log creation errors to stderr but don't fail the application

### Runtime Logging Errors
- If logging fails during runtime, continue normal operation without debug logging
- Attempt to close and recreate log file on write errors
- Log runtime errors to stderr

### Graceful Degradation
- Debug logging failures should never impact normal application functionality
- If debug mode fails to initialize, the application continues in normal mode
- All debug-related errors are non-fatal

## Testing Strategy

### Unit Tests
1. **Debug Log File Creation Tests**
   - Test log file creation in `~/.zamm/logs` directory
   - Test fallback to current directory when logs directory is not writable
   - Test error handling for file operations
   - Test proper file closure

2. **Model Debug Integration Tests**
   - Test Model with debug writer enabled vs disabled
   - Test message dumping in Update method
   - Test normal program execution flow with debug enabled

3. **Message Dumping Tests**
   - Test spew formatting of different message types
   - Test handling of complex nested messages
   - Test performance impact of debug logging

### Integration Tests
1. **End-to-End Debug Flow**
   - Test complete debug logging flow from CLI flag to log file
   - Test log file creation in `~/.zamm/logs`
   - Test message capture during interactive mode

2. **Error Scenarios**
   - Test behavior when log directory is not writable
   - Test behavior when disk is full
   - Test graceful degradation scenarios

### Manual Testing
1. **Interactive Mode Testing**
   - Run interactive mode with debug flag
   - Verify all user interactions are logged
   - Verify log file format and readability

2. **Performance Testing**
   - Verify debug logging doesn't significantly impact performance
   - Test with high message volume scenarios

## Implementation Details

### Log File Naming Convention
- Format: `zamm-debug-YYYY-MM-DD-HH-MM-SS.log`
- Location: `~/.zamm/logs/`
- Each application run creates a new log file to avoid conflicts

### Message Logging Strategy
Debug messages will be logged by:
1. Adding a debug writer field to the Model struct
2. Using `spew.Fdump()` in the Update method to dump each message
3. Preserving all original program behavior and return values

### Configuration Integration
- Add `--debug` flag to root command persistent flags
- Pass debug configuration to interactive mode initialization
- Ensure debug state is available throughout the application lifecycle

### Directory Structure
```
~/.zamm/
├── logs/
│   ├── zamm-debug-2024-01-15-10-30-45.log
│   ├── zamm-debug-2024-01-15-11-15-22.log
│   └── ...
└── ... (existing zamm files)
```

### Performance Considerations
- Spew formatting is only performed when debug writer is not nil
- Log writes are buffered to minimize I/O overhead
- Debug mode adds minimal overhead when disabled
- File handles are properly managed to prevent resource leaks