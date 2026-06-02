/**
 * PM2 Ecosystem Config — POS System
 *
 * Usage:
 *   pm2 start ecosystem.config.js          # start all
 *   pm2 start ecosystem.config.js --env production
 *   pm2 restart all
 *   pm2 stop all
 *   pm2 logs
 *   pm2 monit
 */

const path = require("path");

// Support running from inside pos-backend/ OR from parent directory
const fs = require("fs");
const BACKEND_DIR = __dirname;
const FRONTEND_DIR = path.join(__dirname, "..", "pos-frontend");

module.exports = {
  apps: [
    // ── Go Backend ──────────────────────────────────────────────────────
    {
      name: "pos-backend",
      script: "./bin/api",           // pre-built binary (go build → bin/api)
      cwd: BACKEND_DIR,
      interpreter: "none",           // binary, not a script
      watch: false,
      autorestart: true,
      restart_delay: 3000,
      max_memory_restart: "512M",
      out_file: "/var/log/pm2/pos-backend-out.log",
      error_file: "/var/log/pm2/pos-backend-err.log",
      log_date_format: "YYYY-MM-DD HH:mm:ss Z",
      merge_logs: true,
      env: {
        APP_HOST: "0.0.0.0",
        APP_PORT: "8080",
      },
    },

    // ── Next.js Frontend ─────────────────────────────────────────────────
    {
      name: "pos-frontend",
      script: "npm",
      args: "start",                 // next start (production build)
      cwd: FRONTEND_DIR,
      interpreter: "none",
      watch: false,
      autorestart: true,
      restart_delay: 3000,
      max_memory_restart: "1G",
      out_file: "/var/log/pm2/pos-frontend-out.log",
      error_file: "/var/log/pm2/pos-frontend-err.log",
      log_date_format: "YYYY-MM-DD HH:mm:ss Z",
      merge_logs: true,
      env: {
        NODE_ENV: "production",
        PORT: "3000",
      },
    },
  ],
};
