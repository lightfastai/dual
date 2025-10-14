#!/usr/bin/env node

/**
 * Node.js wrapper for the dual CLI binary
 *
 * This script executes the native dual binary that was downloaded during
 * the postinstall phase. It preserves all arguments, exit codes, and
 * streams stdout/stderr in real-time.
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

/**
 * Determines the binary name based on the current platform
 */
function getBinaryName() {
  return process.platform === 'win32' ? 'dual.exe' : 'dual';
}

/**
 * Locates the dual binary
 */
function getBinaryPath() {
  const binaryName = getBinaryName();
  const binaryPath = path.join(__dirname, binaryName);

  // Check if binary exists
  if (!fs.existsSync(binaryPath)) {
    console.error('[dual] Error: Binary not found at', binaryPath);
    console.error('[dual] The installation may have failed or been corrupted.');
    console.error('[dual] Try reinstalling: npm install @lightfastai/dual');
    console.error('');
    console.error('Alternative installation methods:');
    console.error('  • Homebrew: brew tap lightfastai/tap && brew install dual');
    console.error('  • Direct download: https://github.com/lightfastai/dual/releases');
    process.exit(1);
  }

  return binaryPath;
}

/**
 * Executes the dual binary with the provided arguments
 */
function executeDual() {
  const binaryPath = getBinaryPath();
  const args = process.argv.slice(2); // Remove 'node' and script path

  // Spawn the binary as a child process
  const child = spawn(binaryPath, args, {
    stdio: 'inherit', // Inherit stdin, stdout, stderr for real-time streaming
    env: process.env, // Pass through all environment variables
    windowsHide: false, // Show window on Windows if needed
  });

  // Handle process signals
  // Forward signals to child process
  const signals = ['SIGINT', 'SIGTERM', 'SIGQUIT'];
  signals.forEach((signal) => {
    process.on(signal, () => {
      // Forward signal to child
      if (child.killed === false) {
        child.kill(signal);
      }
    });
  });

  // Handle child process exit
  child.on('error', (err) => {
    console.error('[dual] Error executing binary:', err.message);
    process.exit(1);
  });

  child.on('close', (code, signal) => {
    // Preserve the exit code from the dual binary
    if (signal) {
      // If killed by signal, exit with appropriate code
      process.exit(128 + (signals.indexOf(signal) + 1));
    } else {
      process.exit(code || 0);
    }
  });
}

// Execute dual if this script is run directly
if (require.main === module) {
  executeDual();
}

module.exports = { executeDual, getBinaryPath };
