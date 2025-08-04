# Implementation Plan

- [x] 1. Add debug flag to CLI root command
  - Add `--debug` flag to persistent flags in root command
  - Pass debug flag state to interactive command
  - _Requirements: 2.1, 2.2_

- [x] 2. Create debug log file management utilities
  - Create function to generate debug log file path in `~/.zamm/logs`
  - Implement log file creation with proper error handling
  - Exit with error if logs directory cannot be created or accessed
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 3. Add debug writer field to Model struct
  - Add `debugWriter io.Writer` field to Model struct in interactive.go
  - Update Model initialization to accept debug writer parameter
  - _Requirements: 4.1, 4.2_

- [x] 4. Integrate spew package for message dumping
  - Add spew dependency to go.mod
  - Import spew package in interactive.go
  - _Requirements: 1.1, 1.3_

- [x] 5. Implement message dumping in Update method
  - Add spew.Fdump call at the beginning of Update method
  - Ensure dumping only occurs when debugWriter is not nil
  - Test with different message types to verify output format
  - _Requirements: 1.1, 1.3, 4.3_

- [x] 6. Update interactive command to handle debug mode
  - Modify createInteractiveCommand to accept debug flag
  - Update runInteractiveMode to create debug log file when debug is enabled
  - Pass debug writer to Model initialization
  - Ensure proper cleanup of debug file on program exit
  - _Requirements: 1.2, 1.4, 2.1, 3.1_

