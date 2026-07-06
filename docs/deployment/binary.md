---
title: "Binary Deployment"
description: "Run Tavo as a standalone binary on Linux, macOS, or Windows - no Docker required."
section: deployment
order: 0
---
# Binary Deployment

Tavo ships as a single binary (~15 MB) with the web UI embedded. No runtime dependencies, no containers required.

## Download

Download the latest binary for your platform from the [releases page](https://github.com/jukaza/tavo/releases/latest).

### Linux

    curl -sL https://github.com/jukaza/tavo/releases/latest/download/tavo-linux-amd64.tar.gz | tar xz

For ARM64 (Raspberry Pi, AWS Graviton):

    curl -sL https://github.com/jukaza/tavo/releases/latest/download/tavo-linux-arm64.tar.gz | tar xz

### macOS

    # Apple Silicon (M1/M2/M3)
    curl -sL https://github.com/jukaza/tavo/releases/latest/download/tavo-darwin-arm64.tar.gz | tar xz

    # Intel
    curl -sL https://github.com/jukaza/tavo/releases/latest/download/tavo-darwin-amd64.tar.gz | tar xz

macOS may show a security warning on first run. Allow it in System Settings > Privacy & Security.

### Windows

Download `tavo-windows-amd64.zip` from the [releases page](https://github.com/jukaza/tavo/releases/latest) and extract it.

Windows SmartScreen may show "Windows protected your PC" on first run. Click "More info" then "Run anyway".

## Required Secrets

Tavo needs two secrets to start. Generate them once and keep them safe - changing the encryption key after data is stored will make encrypted values unreadable.

### Linux / macOS

    export TAVO_ADMIN_KEY=$(openssl rand -base64 32)
    export TAVO_ENCRYPTION_KEY=$(openssl rand -base64 32)
    ./tavo

### Windows (PowerShell)

    $env:TAVO_ADMIN_KEY = [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Max 256 }) -as [byte[]])
    $env:TAVO_ENCRYPTION_KEY = [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Max 256 }) -as [byte[]])
    .\tavo.exe

Save these values somewhere secure. You will need the encryption key if you move or restore the database.

**Important:** Use the same `TAVO_ENCRYPTION_KEY` on every start. Changing it after provider API keys are stored makes them unreadable (proxy fails; UI may still show channels as active). Back up the key with the database. See [Local Development](local-dev.md) and [Troubleshooting](../troubleshooting.md).

## First Start

On first start, Tavo creates a SQLite database (`tavo.db`) in the current directory and prints bootstrap credentials:

    ========================================
     BOOTSTRAP COMPLETE - COPY THESE NOW
    ========================================
      API Key:    vl_uk_a3f2...
      Email:      admin@tavo.local
      Password:   <random>
    ========================================

Open http://localhost:8080, log in with the email and password above. These credentials are shown once.

## Configuration

Without a config file, Tavo uses sensible defaults:
- Database: `./tavo.db` (SQLite in current directory)
- Port: 8080
- All features: community edition

For advanced configuration, create a `tavo.yaml` in the same directory:

    server:
      proxy:
        port: 8080

    models:
      - name: my-model
        provider: ollama
        base_url: http://localhost:11434/v1

    settings:
      admin_key: ${TAVO_ADMIN_KEY}
      encryption_key: ${TAVO_ENCRYPTION_KEY}

Tavo auto-discovers `tavo.yaml` in the current directory. Use `--config /path/to/config.yaml` to specify a different location.

## Environment Variables

For config-less operation (no YAML file), these environment variables are supported:

| Variable | Required | Description |
|---|---|---|
| `TAVO_ADMIN_KEY` | First start | Bootstrap admin key (min 32 chars) |
| `TAVO_ENCRYPTION_KEY` | Yes | AES-256-GCM key for encryption |
| `TAVO_DATABASE_DSN` | No | Database path (default: `./tavo.db`) |
| `TAVO_DATABASE_DRIVER` | No | Database driver (default: `sqlite`, alternative: `postgres`) |
| `TAVO_LICENSE` | No | Enterprise license JWT |

## Running as a Service

### Linux (systemd)

Create `/etc/systemd/system/tavo.service`:

    [Unit]
    Description=Tavo LLM Proxy
    After=network.target

    [Service]
    Type=simple
    User=tavo
    WorkingDirectory=/opt/tavo
    ExecStart=/opt/tavo/tavo --config /opt/tavo/tavo.yaml
    Restart=on-failure
    RestartSec=5
    Environment=TAVO_ADMIN_KEY=your-admin-key-here
    Environment=TAVO_ENCRYPTION_KEY=your-encryption-key-here

    [Install]
    WantedBy=multi-user.target

Then:

    sudo systemctl daemon-reload
    sudo systemctl enable --now tavo

### macOS (launchd)

Create `~/Library/LaunchAgents/io.tavo.plist` or use a process manager like `brew services`.

### Windows

Use NSSM (Non-Sucking Service Manager) or Task Scheduler to run `tavo.exe` as a background service.

## Updating

Download the new binary and replace the old one. The database is preserved - no migration steps needed (migrations run automatically on startup).

    # Linux/macOS
    curl -sL https://github.com/jukaza/tavo/releases/latest/download/tavo-linux-amd64.tar.gz | tar xz
    # Restart the service

## Connecting to Ollama

If Ollama runs on the same machine, use `http://localhost:11434/v1` as the base URL. If Tavo runs in Docker but Ollama runs on the host, use `http://host.docker.internal:11434/v1` instead.
