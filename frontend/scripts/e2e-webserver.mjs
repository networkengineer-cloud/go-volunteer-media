import { spawn } from 'node:child_process';
import http from 'node:http';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.resolve(__dirname, '..');
const repoRoot = path.resolve(frontendDir, '..');

function waitForUrl(url, timeoutMs) {
  const deadline = Date.now() + timeoutMs;

  return new Promise((resolve, reject) => {
    const attempt = () => {
      const req = http.get(url, (res) => {
        res.resume();
        if (res.statusCode && res.statusCode >= 200 && res.statusCode < 500) {
          resolve(true);
        } else {
          retry();
        }
      });

      req.on('error', retry);
      req.setTimeout(2000, () => {
        req.destroy(new Error('timeout'));
      });

      function retry() {
        if (Date.now() > deadline) {
          reject(new Error(`Timed out waiting for ${url}`));
          return;
        }
        setTimeout(attempt, 300);
      }
    };

    attempt();
  });
}

function spawnChild(command, args, cwd, env) {
  return spawn(command, args, {
    cwd,
    env,
    stdio: 'inherit',
  });
}

async function isListening(url) {
  try {
    await waitForUrl(url, 1500);
    return true;
  } catch {
    return false;
  }
}

let backendProcess = null;
let frontendProcess = null;

async function main() {
  const backendHealth = 'http://localhost:8080/health';
  const frontendUrl = 'http://localhost:5173';

  const backendRunning = await isListening(backendHealth);
  if (!backendRunning) {
    const env = {
      ...process.env,
      AUTH_RATE_LIMIT_PER_MINUTE: process.env.AUTH_RATE_LIMIT_PER_MINUTE ?? '1000',
    };

    backendProcess = spawnChild('go', ['run', 'cmd/api/main.go'], repoRoot, env);
    await waitForUrl(backendHealth, 120000);
  }

  const frontendRunning = await isListening(frontendUrl);
  if (!frontendRunning) {
    frontendProcess = spawnChild('npm', ['run', 'dev'], frontendDir, process.env);
    await waitForUrl(frontendUrl, 120000);
  }

  // Keep the process alive until Playwright stops the webServer.
  // Playwright sends SIGTERM on shutdown.
  // Monitor child processes and exit if they crash
  const checkInterval = setInterval(() => {
    if (backendProcess && backendProcess.exitCode !== null) {
      clearInterval(checkInterval);
      throw new Error(`Backend exited with code ${backendProcess.exitCode}`);
    }
    if (frontendProcess && frontendProcess.exitCode !== null) {
      clearInterval(checkInterval);
      throw new Error(`Frontend exited with code ${frontendProcess.exitCode}`);
    }
  }, 1000);

  // Wait indefinitely for SIGTERM (event-driven approach)
  await new Promise(() => {});
}

function shutdown() {
  if (frontendProcess) frontendProcess.kill('SIGTERM');
  if (backendProcess) backendProcess.kill('SIGTERM');
}

process.on('SIGINT', () => {
  shutdown();
  process.exit(0);
});

process.on('SIGTERM', () => {
  shutdown();
  process.exit(0);
});

main().catch((err) => {
  // eslint-disable-next-line no-console
  console.error(err);
  shutdown();
  process.exit(1);
});
