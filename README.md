```
   ____ _            _           _____ _               
  / ___| | __ _  ___(_) ___ _ __|  ___| | _____      __
 | |  _| |/ _` |/ __| |/ _ \ '__| |_  | |/ _ \ \ /\ / /
 | |_| | | (_| | (__| |  __/ |  |  _| | | (_) \ V  V / 
  \____|_|\__,_|\___|_|\___|_|  |_|   |_|\___/ \_/\_/  
                                                       
```

# GlacierFlow

A lightweight toolkit for managing local AI inference servers (llama.cpp) on Linux, with hot-swap model presets, embedding server support, and automatic container orchestration. Includes Web UI, TUI and pi.dev container for software development.

---

## What It Does

GlacierFlow runs llama.cpp inside Docker and lets you switch between model presets on the fly — no manual container restarts needed. A background daemon watches your config and applies changes automatically.

**Current features:**
- llama.cpp inference server managed via Docker Compose
- Hot-swap between inference presets (YAML configs)
- Automatic container restarts when preset changes
- Preset loading time tracking & status monitoring
- Example configs for various GPU setups
- Docker-based code generation assistant (pi.dev) with automation workflows
- Multi-model benchmarking suite
- Web-based UI for preset management and status monitoring
- Embedding server (llama.cpp) for vector embeddings, toggleable via override config
- FIFO proxy for serializing chat requests (prevents stream cut-offs during model hot-swap)

---

## Prerequisites

- Linux (tested on Arch/Manjaro)
- Docker & Docker Compose
- GPU drivers (Vulkan for AMD/NVIDIA)
- Bash, curl, grep, awk, md5sum, and other coreutils

---

## Quick Start

### 1. Clone & prepare

```bash
git clone https://github.com/howanski/GlacierFlow.git
cd GlacierFlow
```

### 2. Copy a preset

Pick an example preset and copy it to your local presets directory:

```bash
cp data/shared/inference_presets/examples/vulkan_8gb_gemma4_e4b_q5.yml \
   data/shared/inference_presets/local/
```

### 3. Configure environment

Create a `.env` file with your paths:

```env
GF_LLAMA_MEMORY_LIMIT=50g
GF_LLAMA_SERVER_VERSION=server-vulkan
GF_MODELS_DIRECTORY=/path/to/your/models
GF_PI_DEV_WORKDIR=/path/to/coder/workdir
GF_WEB_UI_PORT=8090
```

### 4. Start the daemon

Run the background service to spin up the inference server:

```bash
cd scripts
./glacierflow_host_daemon
```
Personally I've just added it to my crontab @reboot via thin wrapper in case it blows up (and fancy blocker when I launch Steam)

For a one-shot check (no daemon loop):

```bash
cd scripts
./glacierflow_host_daemon --once
```

### 5. Switch presets

List available presets:

```bash
cd scripts
./glacierflow_inference_select_preset
```

Load a preset by name or hash:

```bash
cd scripts
./glacierflow_inference_select_preset vulkan_8gb_gemma4_e4b_q5
./glacierflow_inference_select_preset <md5hash>
```

---

## Web UI

GlacierFlow includes a web-based management interface for monitoring and controlling the inference server and presets.

### Setup

Make sure `GF_WEB_UI_PORT` is set in your `.env` file:

```env
GF_WEB_UI_PORT=8090
```

### Usage

Start the daemon first:

```bash
cd scripts
./glacierflow_host_daemon
```

Open `http://localhost:8090` in your browser. The UI provides:

- **Preset management** — list, select, and track active presets
- **Status monitoring** — (almost) real-time server status (RUNNING, LOADING, STOPPED, etc.)
- **Lock/unlock** — to force keep inference server off when in locked state

The web UI communicates directly with the shared data directory, so it stays in sync with the daemon and CLI tools.

---

## Preset Files

Preset files are Docker Compose YAML snippets that override the base `compose.yml`. They define:

- **command**: llama.cpp server arguments (model path, context size, GPU layers, etc.)
- **devices**: GPU device passthrough (e.g., `/dev/dri/renderD128` for AMD)
- **volumes**: Model directory mount paths

Preset names are derived from their filename (`.yml` extension stripped). You can reference them by full name or by MD5 hash.

Example presets live in `inference_presets/examples/` (tracked). Your custom presets go in `inference_presets/local/` (gitignored).

---

## Benchmarking

GlacierFlow includes a benchmark runner that measures throughput (tokens/sec) across multiple models and problem types:

```bash
# Show results table (presets × benchmarks)
cd scripts
./glacierflow_benchmark

# Run benchmarks for the currently loaded model
cd scripts
./glacierflow_benchmark run
```

The runner:
- Executes each benchmark 3 times and reports the average TPS
- Skips benchmarks already completed for a given model hash
- Stores results in `data/shared/benchmarks/<benchmark_name>.txt` (one line per model hash)
- Requires the inference server to be running (`RUNNING` status)

### Benchmark problems

| File | Type | Description |
|------|------|-------------|
| `c1.json` | SQL | Generate a MySQL query from a schema |
| `c2.json` | Bash | Write a file-sorting script |
| `c3.json` | PHP | Optimize a user-processing loop |
| `p1.json` | Creative | Write a ~100-word travel description |


---

## Status

The daemon tracks the server state in `data/shared/inference_status.txt`:

| Status | Meaning |
|--------|---------|
| `RUNNING` | Server is up and responding (HTTP 200) |
| `LOADING` | Server container started, model still loading |
| `STOPPED` | Server is down |
| `LAUNCHING` | Docker compose is pulling/starting |
| `LOCKED` | Server is locked (inference lock file present) |
| `NOT CONFIGURED` | No active config found |
| `UMM...` | Unexpected response |

---

## FIFO Proxy

GlacierFlow includes a Go-based FIFO (first-in-first-out) proxy that sits between clients and the llama.cpp inference server. It serializes `/v1/chat/` requests so they are processed strictly one at a time, preventing SSE stream cut-offs when the model is hot-swapped mid-inference.

### How It Works

- **Serialized routing**: `/v1/chat/` requests are queued and dispatched one-by-one to the backend
- **Parallel routing**: All other routes (health checks, non-chat endpoints) are proxied in parallel via standard reverse proxy
- **Auto-rebuild**: The entrypoint detects source changes and rebuilds the Go binary automatically
- **Restart loop**: If the proxy crashes, it restarts after a 2-second delay

### Configuration

The proxy runs on port **8070** (container port 8080). It is started automatically by the daemon — no manual setup required.

If you prefer to bypass the proxy entirely, uncomment the `ports` section in `docker/llama_cpp/compose.yml` and point your clients directly at port 8080 (Or create preset with ports section override).

---

## Embedding Server

GlacierFlow includes a configurable llama.cpp embedding server for generating vector embeddings. It is disabled by default and can be enabled by creating an override config.

### Setup

1. **Enable the server** by creating `docker/llama_embeddings/compose.override.yml` with your desired command:

```yaml
services:
  glacierflow-embeddings:
    command: >
      -hf ggml-org/embeddinggemma-300M-GGUF
      --embeddings
```

2. **Configure the port** in your `.env` file (defaults to 8060):

```env
GF_EMBEDDING_PORT=8060
```

The embedding server starts automatically alongside the main daemon when the override config is present.

### Usage

Once running, the embedding server exposes an OpenAI-compatible API at `http://localhost:8060/v1/embeddings`.

---

## Coder: Code Generation Assistant

GlacierFlow ships with a Dockerized [pi.dev](https://pi.dev) development environment for AI-assisted code generation. It runs inside a full-featured dev container with all common languages and tools pre-installed.

### Setup

1. **Configure the model provider** in `data/pi_dev/config/agent/models.json`:

```json
{
  "providers": {
    "glacierflow-llamacpp": {
      "baseUrl": "http://glacierflow-proxy:8080/v1",
      "api": "openai-completions",
      "apiKey": "none",
      "models": [
        { "id": "glacierflow-llamacpp" }
      ]
    }
  }
}
```

> **Note**: By default this points through the FIFO proxy (`glacierflow-proxy`) so chat requests are serialized. If you bypass the proxy, change the URL to `http://glacierflow-llamacpp:8080/v1`.

2. **Set your inference preset** — the pi.dev container by default uses the model you configured with `glacierflow_inference_select_preset` script.

### Usage

Run the management script to start, stop, or attach to the container:

```bash
cd scripts
./glacierflow_pi_code
```

This presents an interactive menu:

| Key | Action |
|-----|--------|
| `U` | Start the pi.dev container |
| `D` | Stop the container |
| `A` | Attach to the running container |
| `R` | Rebuild / hard reset the container |
| `X` | Exit |

### Inside the Container

Once inside, the `start` script provides an interactive menu with several modes:

| Key | Action |
|-----|--------|
| `N` | New session |
| `C` | Continue last session |
| `S` | Select session |
| `A` | **Automation** - AI-driven development workflow |
| `K` | Kick-off menu (Code Review, Improvements, Readme) |
| `B` | Bash shell |
| `X` | Exit / detach |

#### Automation Mode

The automation menu provides AI-driven development workflows:

| Key | Action |
|-----|--------|
| `S` | Prepare/update SKETCH file (high-level goals) |
| `P` | Convert SKETCH → TODO (non-interactive planner) |
| `Q` | Convert SKETCH → TODO (interactive planner) |
| `T` | Prepare/update TODO file (exact implementation steps) |
| `R` | Run automatic development from TODO |
| `C` | Run automatic development with internal critic loop |

The automation scripts (`data/pi_dev/scripts/builtin/`) guide the AI through structured development: planning, coding, and critiquing iterations.

### Container Details

- **Image**: Alpine-based with bash, git, npm, vim, rust, cargo, php, go, openjdk17, gradle, android-tools, and composer
- **User**: Runs as non-root (`pi_dev`, UID 1000)
- **Volume mounts**:
  - Project root → `/pi_dev` (read-only)
  - Config → `~/.pi` (read-write)
  - Scripts → `/pi_dev_scripts` (read-write, for builtin automation scripts)
  - Workdir → configurable via `GF_PI_DEV_WORKDIR` env var (read-write)

---

## Remotes

- [GitHub](https://github.com/howanski/GlacierFlow)
- [Codeberg](https://codeberg.org/howanski/GlacierFlow)
- [Gitea (intranet)](https://gitea.howan.ski/howanski/GlacierFlow)

---

## Roadmap

- [x] **coder** - Code generation assistant integration
- [x] **coder automation** - Interactive menu, SKETCH→TODO pipeline, auto-programmer, auto-critic
- [x] **webui** - Web-based UI for preset management
- [x] **embeddings** - llama.cpp embedding server
- [x] **fifo-proxy** - Go-based FIFO proxy for serializing chat requests
- [ ] **agent** - Autonomous agent mode

---

## License

GlacierFlow itself is a collection of custom scripts — no license applies.
Third-party software (llama.cpp, Docker, pi.dev etc.) is subject to their respective licenses.
