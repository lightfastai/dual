#!/usr/bin/env node

/**
 * Worker Service - Multi-Worktree Example
 *
 * This service demonstrates a background worker that uses the same
 * environment configuration as the web and API services.
 */

const path = require('path');

// Load environment variables from multiple locations
// 1. Service-specific .env file (apps/worker/.env)
// 2. Root .env.local file (../../.env.local)
require('dotenv').config({ path: path.join(__dirname, '.env') });
require('dotenv').config({ path: path.join(__dirname, '../../.env.local') });

// Get port from environment (workers typically don't use ports, but we include it for consistency)
const PORT = process.env.PORT || process.env.PORT_WORKER || 3003;

// Get context information
const CONTEXT = process.env.DUAL_CONTEXT || 'unknown';
const BASE_PORT = process.env.DUAL_BASE_PORT || 'unknown';

// Job queue simulation
let jobCount = 0;

// Simulate processing jobs
function processJob() {
  jobCount++;
  const jobId = `job-${jobCount}`;

  console.log(`[${new Date().toISOString()}] Processing ${jobId} in context "${CONTEXT}"`);

  // Simulate some work
  const processingTime = Math.floor(Math.random() * 2000) + 1000;

  setTimeout(() => {
    console.log(`[${new Date().toISOString()}] Completed ${jobId} (took ${processingTime}ms)`);
  }, processingTime);
}

// Main worker function
function startWorker() {
  console.log('='.repeat(60));
  console.log('Worker Service Started');
  console.log('='.repeat(60));
  console.log(`Context:      ${CONTEXT}`);
  console.log(`Port:         ${PORT} (not used for HTTP)`);
  console.log(`Base Port:    ${BASE_PORT}`);
  console.log(`Environment:  ${process.env.NODE_ENV || 'development'}`);
  console.log(`Database:     ${process.env.DATABASE_NAME || 'not configured'}`);
  console.log(`Redis DB:     ${process.env.REDIS_DB || 'not configured'}`);
  console.log('='.repeat(60));
  console.log('');
  console.log('Worker is now processing jobs...');
  console.log('Press Ctrl+C to stop the worker');
  console.log('');

  // Process a job every 5 seconds
  setInterval(() => {
    processJob();
  }, 5000);

  // Process first job immediately
  processJob();
}

// Start the worker
startWorker();

// Graceful shutdown
let isShuttingDown = false;

function shutdown() {
  if (isShuttingDown) {
    return;
  }
  isShuttingDown = true;

  console.log('');
  console.log('Shutting down worker gracefully...');
  console.log(`Processed ${jobCount} jobs in total`);
  console.log('Goodbye!');

  // Give some time for cleanup
  setTimeout(() => {
    process.exit(0);
  }, 500);
}

process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);
