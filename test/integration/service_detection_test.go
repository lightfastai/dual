package integration

// All service detection tests in this file were removed because they were entirely focused
// on port detection, querying, and command wrapper functionality which has been removed.
// The worktree lifecycle manager no longer manages ports or requires service auto-detection
// for most operations.
//
// Tests removed:
// - TestServiceAutoDetection: port querying from different directories
// - TestServiceDetectionLongestMatch: port detection with nested services
// - TestServiceDetectionWithSymlinks: port detection with symlinks
// - TestServiceDetectionMultipleServices: port calculation with multiple services
// - TestServiceDetectionErrorMessages: port querying error messages
// - TestServiceDetectionWithCommandWrapper: command wrapper mode with PORT injection
