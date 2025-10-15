#!/usr/bin/env node

/**
 * Web Service - Multi-Worktree Example
 *
 * This service demonstrates how environment variables (especially PORT)
 * are loaded from the worktree-specific .env files created by dual hooks.
 */

const express = require('express');
const path = require('path');

// Load environment variables from multiple locations
// 1. Service-specific .env file (apps/web/.env)
// 2. Root .env.local file (../../.env.local)
require('dotenv').config({ path: path.join(__dirname, '.env') });
require('dotenv').config({ path: path.join(__dirname, '../../.env.local') });

const app = express();

// Get port from environment (set by dual hooks)
// Falls back to 3001 if PORT_WEB not set (shouldn't happen in dual-managed worktrees)
const PORT = process.env.PORT || process.env.PORT_WEB || 3001;

// Get context information
const CONTEXT = process.env.DUAL_CONTEXT || 'unknown';
const BASE_PORT = process.env.DUAL_BASE_PORT || 'unknown';

// Simple route to show service information
app.get('/', (req, res) => {
  res.json({
    service: 'web',
    context: CONTEXT,
    port: PORT,
    basePort: BASE_PORT,
    message: `Web service running in context "${CONTEXT}"`,
    environment: {
      nodeEnv: process.env.NODE_ENV,
      apiUrl: process.env.API_URL,
      webUrl: process.env.WEB_URL,
    },
    timestamp: new Date().toISOString(),
  });
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'web',
    context: CONTEXT,
    port: PORT,
  });
});

// Start the server
app.listen(PORT, () => {
  console.log('='.repeat(60));
  console.log('Web Service Started');
  console.log('='.repeat(60));
  console.log(`Context:     ${CONTEXT}`);
  console.log(`Port:        ${PORT}`);
  console.log(`Base Port:   ${BASE_PORT}`);
  console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
  console.log(`URL:         http://localhost:${PORT}`);
  console.log('='.repeat(60));
  console.log('');
  console.log('Try these endpoints:');
  console.log(`  GET http://localhost:${PORT}/        - Service info`);
  console.log(`  GET http://localhost:${PORT}/health  - Health check`);
  console.log('');
  console.log('Press Ctrl+C to stop the server');
  console.log('');
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('\nReceived SIGTERM, shutting down gracefully...');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('\nReceived SIGINT, shutting down gracefully...');
  process.exit(0);
});
