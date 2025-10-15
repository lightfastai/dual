#!/usr/bin/env node

/**
 * API Service - Multi-Worktree Example
 *
 * This service demonstrates how environment variables (especially PORT)
 * are loaded from the worktree-specific .env files created by dual hooks.
 */

const express = require('express');
const path = require('path');

// Load environment variables from multiple locations
// 1. Service-specific .env file (apps/api/.env)
// 2. Root .env.local file (../../.env.local)
require('dotenv').config({ path: path.join(__dirname, '.env') });
require('dotenv').config({ path: path.join(__dirname, '../../.env.local') });

const app = express();

// Parse JSON request bodies
app.use(express.json());

// Get port from environment (set by dual hooks)
// Falls back to 3002 if PORT_API not set (shouldn't happen in dual-managed worktrees)
const PORT = process.env.PORT || process.env.PORT_API || 3002;

// Get context information
const CONTEXT = process.env.DUAL_CONTEXT || 'unknown';
const BASE_PORT = process.env.DUAL_BASE_PORT || 'unknown';

// Simple route to show service information
app.get('/', (req, res) => {
  res.json({
    service: 'api',
    context: CONTEXT,
    port: PORT,
    basePort: BASE_PORT,
    message: `API service running in context "${CONTEXT}"`,
    environment: {
      nodeEnv: process.env.NODE_ENV,
      databaseUrl: process.env.DATABASE_URL,
      databaseName: process.env.DATABASE_NAME,
      redisUrl: process.env.REDIS_URL,
      redisDb: process.env.REDIS_DB,
    },
    timestamp: new Date().toISOString(),
  });
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'api',
    context: CONTEXT,
    port: PORT,
  });
});

// Example API endpoint
app.get('/api/users', (req, res) => {
  res.json({
    users: [
      { id: 1, name: 'Alice', context: CONTEXT },
      { id: 2, name: 'Bob', context: CONTEXT },
    ],
    meta: {
      context: CONTEXT,
      port: PORT,
    },
  });
});

// Example POST endpoint
app.post('/api/users', (req, res) => {
  const user = req.body;
  res.status(201).json({
    message: 'User created',
    user: {
      ...user,
      id: Math.floor(Math.random() * 1000),
      context: CONTEXT,
    },
  });
});

// Start the server
app.listen(PORT, () => {
  console.log('='.repeat(60));
  console.log('API Service Started');
  console.log('='.repeat(60));
  console.log(`Context:      ${CONTEXT}`);
  console.log(`Port:         ${PORT}`);
  console.log(`Base Port:    ${BASE_PORT}`);
  console.log(`Environment:  ${process.env.NODE_ENV || 'development'}`);
  console.log(`Database:     ${process.env.DATABASE_NAME || 'not configured'}`);
  console.log(`URL:          http://localhost:${PORT}`);
  console.log('='.repeat(60));
  console.log('');
  console.log('Try these endpoints:');
  console.log(`  GET  http://localhost:${PORT}/           - Service info`);
  console.log(`  GET  http://localhost:${PORT}/health     - Health check`);
  console.log(`  GET  http://localhost:${PORT}/api/users  - List users`);
  console.log(`  POST http://localhost:${PORT}/api/users  - Create user`);
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
