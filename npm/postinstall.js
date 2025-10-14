#!/usr/bin/env node

/**
 * Postinstall script for @lightfastai/dual npm package
 *
 * This script downloads the appropriate dual binary from GitHub Releases
 * based on the user's platform and architecture. It follows the pattern
 * used by esbuild, @swc/core, and biome for distributing native binaries
 * through npm.
 *
 * Supported platforms:
 * - darwin (macOS): x64, arm64
 * - linux: x64, arm64
 * - win32 (Windows): x64, arm64
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const { promisify } = require('util');
const { pipeline } = require('stream');
const zlib = require('zlib');
const tar = require('tar');

const streamPipeline = promisify(pipeline);

// Get package version from package.json
const packageJson = require('./package.json');
const VERSION = packageJson.version;

/**
 * Maps Node.js platform identifiers to dual's platform naming
 */
const PLATFORM_MAP = {
  darwin: 'Darwin',
  linux: 'Linux',
  win32: 'Windows',
};

/**
 * Maps Node.js architecture identifiers to dual's architecture naming
 */
const ARCH_MAP = {
  x64: 'x86_64',
  arm64: 'arm64',
};

/**
 * Determines the correct binary filename based on platform
 */
function getBinaryName(platform) {
  return platform === 'win32' ? 'dual.exe' : 'dual';
}

/**
 * Constructs the download URL for the dual binary
 * Format: https://github.com/lightfastai/dual/releases/download/v{version}/dual_{platform}_{arch}.tar.gz
 */
function getDownloadUrl(platform, arch) {
  const mappedPlatform = PLATFORM_MAP[platform];
  const mappedArch = ARCH_MAP[arch];

  if (!mappedPlatform || !mappedArch) {
    throw new Error(
      `Unsupported platform/architecture: ${platform}/${arch}\n` +
      `Supported platforms: ${Object.keys(PLATFORM_MAP).join(', ')}\n` +
      `Supported architectures: ${Object.keys(ARCH_MAP).join(', ')}`
    );
  }

  return `https://github.com/lightfastai/dual/releases/download/v${VERSION}/dual_${mappedPlatform}_${mappedArch}.tar.gz`;
}

/**
 * Downloads a file from the given URL with redirect following
 */
function downloadFile(url) {
  return new Promise((resolve, reject) => {
    console.log(`[dual] Downloading from: ${url}`);

    https.get(url, { headers: { 'User-Agent': 'dual-npm-installer' } }, (response) => {
      // Follow redirects (GitHub releases redirect to CDN)
      if (response.statusCode === 301 || response.statusCode === 302) {
        const redirectUrl = response.headers.location;
        console.log(`[dual] Following redirect to: ${redirectUrl}`);
        return downloadFile(redirectUrl).then(resolve).catch(reject);
      }

      if (response.statusCode !== 200) {
        reject(new Error(
          `Failed to download dual binary: HTTP ${response.statusCode}\n` +
          `URL: ${url}\n` +
          `This may indicate:\n` +
          `  1. The release v${VERSION} doesn't exist on GitHub\n` +
          `  2. Network connectivity issues\n` +
          `  3. GitHub API rate limiting\n\n` +
          `Try installing dual directly:\n` +
          `  brew tap lightfastai/tap && brew install dual`
        ));
        return;
      }

      resolve(response);
    }).on('error', (err) => {
      reject(new Error(
        `Network error while downloading dual binary: ${err.message}\n` +
        `Please check your internet connection and try again.`
      ));
    });
  });
}

/**
 * Extracts the binary from a .tar.gz archive
 */
async function extractBinary(response, destPath, binaryName) {
  const gunzip = zlib.createGunzip();
  const extract = tar.extract({
    cwd: path.dirname(destPath),
    filter: (filePath) => {
      // Only extract the binary file, ignore other files in the archive
      return path.basename(filePath) === binaryName;
    },
  });

  try {
    await streamPipeline(response, gunzip, extract);
  } catch (err) {
    throw new Error(
      `Failed to extract dual binary: ${err.message}\n` +
      `The downloaded archive may be corrupted. Please try again.`
    );
  }

  // The binary is extracted with its original name, rename if needed
  const extractedPath = path.join(path.dirname(destPath), binaryName);
  if (extractedPath !== destPath) {
    fs.renameSync(extractedPath, destPath);
  }
}

/**
 * Makes the binary executable (Unix-like systems only)
 */
function makeExecutable(filePath) {
  if (process.platform !== 'win32') {
    try {
      fs.chmodSync(filePath, 0o755);
    } catch (err) {
      throw new Error(
        `Failed to make binary executable: ${err.message}\n` +
        `Try running manually: chmod +x ${filePath}`
      );
    }
  }
}

/**
 * Main installation function
 */
async function install() {
  const platform = process.platform;
  const arch = process.arch;
  const binaryName = getBinaryName(platform);
  const binDir = path.join(__dirname, 'bin');
  const binaryPath = path.join(binDir, binaryName);

  console.log(`[dual] Installing for ${platform}-${arch}`);

  // Check if binary already exists (useful for local development)
  if (fs.existsSync(binaryPath)) {
    console.log(`[dual] Binary already exists at ${binaryPath}`);
    makeExecutable(binaryPath);
    console.log('[dual] Installation complete!');
    return;
  }

  try {
    // Ensure bin directory exists
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // Get download URL
    const downloadUrl = getDownloadUrl(platform, arch);

    // Download and extract binary
    const response = await downloadFile(downloadUrl);
    await extractBinary(response, binaryPath, binaryName);

    // Verify binary exists
    if (!fs.existsSync(binaryPath)) {
      throw new Error(
        'Binary extraction completed but file not found. ' +
        'This may indicate an issue with the archive structure.'
      );
    }

    // Make executable
    makeExecutable(binaryPath);

    console.log('[dual] Installation complete!');
    console.log(`[dual] Binary installed at: ${binaryPath}`);

  } catch (err) {
    console.error('\n[dual] Installation failed:');
    console.error(err.message);
    console.error('\nAlternative installation methods:');
    console.error('  • Homebrew: brew tap lightfastai/tap && brew install dual');
    console.error('  • Direct download: https://github.com/lightfastai/dual/releases');
    console.error('  • Build from source: go install github.com/lightfastai/dual/cmd/dual@latest');
    process.exit(1);
  }
}

// Run installation if this script is executed directly
// (not when required as a module)
if (require.main === module) {
  install().catch((err) => {
    console.error('[dual] Unexpected error:', err);
    process.exit(1);
  });
}

module.exports = { install };
