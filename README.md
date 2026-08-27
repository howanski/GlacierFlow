```
   ____ _            _           _____ _               
  / ___| | __ _  ___(_) ___ _ __|  ___| | _____      __
 | |  _| |/ _` |/ __| |/ _ \ '__| |_  | |/ _ \ \ /\ / /
 | |_| | | (_| | (__| |  __/ |  |  _| | | (_) \ V  V / 
  \____|_|\__,_|\___|_|\___|_|  |_|   |_|\___/ \_/\_/  
                                                       
```

# GlacierFlow

A lightweight toolkit for managing local AI inference servers (llama.cpp / stable-diffusion.cpp / audio.cpp) on Linux, with hot-swap model presets, embedding server support, and automatic container orchestration. Includes Web UI, TUI and pi.dev container for software development.

---

## What It Does

GlacierFlow runs llama.cpp inside Docker and lets you switch between model presets on the fly — no manual container restarts needed. A background daemon watches your config and applies changes automatically.

**Current features:**
- llama.cpp / stable-diffusion.cpp / audio.cpp inference server managed via Docker Compose
- Hot-swap between inference presets (YAML configs)
- Automatic container restarts when preset changes
- Preset loading time tracking & status monitoring
- Example configs for various GPU setups
- Docker-based code generation assistant (pi.dev) with automation workflows
- Hermes AI agent container with web dashboard and kanban task support
- Multi-model benchmarking suite
- GPU layers autotune — binary search for optimal VRAM offload with stability testing and quick benchmark
- Web-based UI for preset management and status monitoring
- Embedding server (llama.cpp) for vector embeddings, toggleable via override config
- FIFO proxy for serializing chat requests (prevents stream cut-offs during model hot-swap)
- Caddy reverse proxy with SSL/TLS termination and basic auth for Web UI, pi.dev TTYD, Hermes TTYD and VS Code
- User/group ID customization via `GF_USER_ID` / `GF_USER_GROUP_ID` environment variables (affects all containers)

---

## Prerequisites

- Linux (tested on Arch/Manjaro)
- Docker & Docker Compose
- GPU drivers (Vulkan for AMD/NVIDIA)
- Bash, curl, git, grep, awk, md5sum, openssl, unzip, and other coreutils

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
cp data/shared/inference_presets/examples/vulkan_8gb_gemma4_e4b_q4.yml \
   data/shared/inference_presets/local/
```

By default a self-signed certificate is auto-generated on first run. To use your own certificates, place `cert.pem` and `key.pem` in `data/ssl_certs/` before starting the daemon.

```bash
# Generate self-signed cert manually (daemon does this automatically if files are missing)
openssl req -x509 -newkey rsa:4096 -keyout data/ssl_certs/key.pem -out data/ssl_certs/cert.pem -days 3650 -nodes
```

### 3. Configure environment

Create a `.env` file with your paths:

```env
GF_HERMES_AUTOSTART=0 # set to 1 to auto-start Hermes container with the daemon
GF_HERMES_TTYD_PORT=7682
GF_HERMES_WEB_PORT=7683
GF_HERMES_WORKDIR=/path/to/hermes/workdir
GF_USER_GROUP_ID=1000 # group ID for all containers (defaults to 1000)
GF_USER_ID=1000 # user ID for all containers (defaults to 1000)
GF_AUDIO_CPP_ENABLE=0 # set to 1 to auto-clone/build audio.cpp binaries on daemon start (CPU-heavy)
GF_STABLE_DIFFUSION_ENABLE=0 # set to 1 to auto-download/update stable-diffusion.cpp binaries on daemon start
GF_VSCODE_AUTOSTART=0 # set to 1 to auto-start VS Code container with the daemon
GF_VSCODE_WEB_PORT=7684
GF_HTTP_PASSWORD=glacierflow
GF_HTTP_USER=glacierflow
GF_LLAMA_MEMORY_LIMIT=50g
GF_LLAMA_SERVER_VERSION=server-vulkan
GF_LOG_STATS=0 # set to 1 to enable performance stats logging
GF_MODELS_DIRECTORY=/path/to/your/models
GF_PI_DEV_TTYD_PORT=7681
GF_PI_DEV_WORKDIR=/path/to/coder/workdir
GF_PROXY_DUMP_REQUESTS=0 # set to 1 to dump request/response pairs to JSON files
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
./glacierflow_inference_select_preset vulkan_8gb_gemma4_e4b_q4
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

Open `https://localhost:8090` in your browser. The UI provides:

- **Preset management** — list, select, and track active presets
- **Status monitoring** — (almost) real-time server status (RUNNING, LOADING, STOPPED, etc.)
- **Lock/unlock** — to force keep inference server off when in locked state
- **Stats Viewer** — Chart.js-based visualization of inference performance metrics (fill TPS, gen TPS, draft acceptance rate)
- **pi.dev httpd** — direct link to the TTYD web terminal for the code generation assistant
- **hermes httpd** — direct link to the TTYD web terminal for the Hermes agent
- **hermes Web UI** — direct link to the Hermes dashboard
- **VS Code** — direct link to the code-server instance


The web UI communicates directly with the shared data directory, so it stays in sync with the daemon and CLI tools.

---

## Preset Files

Preset files are Docker Compose YAML snippets that override the base `compose.yml`. They define:

- **command**:inference server arguments (model path, context size, GPU layers, diffusion models, etc.)
- **devices**: GPU device passthrough (e.g., `/dev/dri` for AMD)
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

# Run benchmarks for the currently loaded model (default 3 passes per benchmark)
cd scripts
./glacierflow_benchmark run

# Run benchmarks with a custom number of passes per benchmark
cd scripts
./glacierflow_benchmark run 5
```

The runner:
- Executes each benchmark N times and reports the average TPS (default 3; set via the `passes` argument or the interactive prompt)
- Skips benchmarks already completed for a given model hash
- Stores results in `data/shared/benchmarks/<benchmark_name>.txt` (one line per model hash)
- Requires the inference server to be running (`RUNNING` status)

### Benchmark problems

| File | Type | Description |
|------|------|-------------|
| `c1.json` | SQL | Generate a MySQL query from a schema |
| `c2.json` | Bash | Write a file-sorting script |
| `c3.json` | PHP | Optimize a user-processing loop |
| `c4.json` | Python | Add token-bucket rate limiting to an API gateway |
| `c5.json` | TypeScript | Refactor a checkout service into a typed module |
| `c6.json` | Go | Fix a data race in a shared counter |

---

## GPU Layers Autotune

GlacierFlow includes a GPU layers autotune script that finds the optimal number of layers to offload to the GPU. It uses binary search to locate the highest layer count that still passes a context stability test, then optionally runs a quick benchmark across all layers.

```bash
cd scripts
./glacierflow_autotune_gpu_layers
```
### How It Works

- **Stability test** — runs `llama-benchy` (via `uvx`) with a given context size (`--pp`) and checks whether the inference server can process the prompt without OOM or errors
- **Binary search** — starts from the current `--gpu-layers` value and searches up or down to find the flip point (highest passing layer)
- **Quick benchmark** — sends 3 requests using the `c1.json` preset and averages the predicted TPS, stepping down one layer at a time

The script modifies `--gpu-layers` in the compose source file and restarts the inference server between each test. It requires the daemon to be running so it can manage the server lifecycle.

### Caveats

- The autotune script works on currently loaded preset and does not support llama's router mode
- The preset file must contain a `--gpu-layers` argument in one line for the script to work


---

## Stable Diffusion

GlacierFlow supports running [stable-diffusion.cpp](https://github.com/leejet/stable-diffusion.cpp) for image generation, replacing llama.cpp inference server. The daemon can automatically download and update the binaries on start.

### Setup

1. **Enable in `.env`** — set `GF_STABLE_DIFFUSION_ENABLE=1` to have the daemon fetch/update stable-diffusion.cpp binaries automatically on each startup:

```env
GF_STABLE_DIFFUSION_ENABLE=1
```

The source URL is configurable via `GF_STABLE_DIFFUSION_SOURCE` in `.env`. The default points to a Vulkan build. Binaries are extracted into `data/stable_diffusion/binaries/`, and the current version is tracked in `data/stable_diffusion/current_source.txt`.

2. **Pick a preset** — example Stable Diffusion presets live in `data/shared/inference_presets/examples/`. Copy one to your local directory:

```bash
cp data/shared/inference_presets/examples/vulkan_8gb_stable_diffusion_realistic_stock_photo_v20.yml \
   data/shared/inference_presets/local/
```

Select it as usual via the preset switcher or Web UI.

3. **Restart the daemon** — the binaries will be downloaded on next boot and the inference server will restart with the new preset.

### Example Presets

| Preset | Model | Notes |
|--------|-------|-------|
| `vulkan_8gb_stable_diffusion_realistic_stock_photo_v20.yml` | realisticStockPhoto v2.0 | SD 1.5 model, Vulkan + CPU offload |
| `vulkan_8gb_stable_diffusion_ideogram_4.yml` | ideogram-4 GGUF | Includes LLM VAE pipeline with Qwen3-VL, Vulkan + CPU offload |

### How It Works

When `GF_STABLE_DIFFUSION_ENABLE=1`, the daemon calls `data/stable_diffusion/update_source.sh` during startup. The script:
- Checks if the binary source URL has changed (compared against `current_source.txt`)
- Downloads and extracts the release archive into `binaries/`
- Overwrites any existing installation

The preset YAML files use a Dockerfile at `data/stable_diffusion/docker/Dockerfile` that installs Vulkan drivers and runs `/app/sd-server` as entrypoint. The inference server port (8080) is shared.

---

## Audio.cpp

GlacierFlow supports running [audio.cpp](https://github.com/0xShug0/audio.cpp) for audio model inference, replacing llama.cpp inference server. The daemon can automatically clone and build the source on start.

### Setup

1. **Enable in `.env`** — set `GF_AUDIO_CPP_ENABLE=1` to have the daemon clone/build audio.cpp automatically on each startup:

```env
GF_AUDIO_CPP_ENABLE=1
```

The source is cloned from `https://github.com/0xShug0/audio.cpp.git` into `~/.audio.cpp-src` (configurable via `GF_AUDIO_CPP_SRC_DIR`). The build is CPU-heavy, so you may prefer to run `data/audio_cpp/update_source.sh` manually with `GF_AUDIO_CPP_ENABLE=0`. The current build version (git commit) is tracked in `data/audio_cpp/last_build.txt` — the build is skipped when the source is unchanged.

2. **Place models** — audio.cpp models (GGUF) go into the `AUDIO_CPP` subdirectory of your models directory (`GF_MODELS_DIRECTORY/AUDIO_CPP/`). Models are available at [audio-cpp/audio.cpp-gguf](https://huggingface.co/audio-cpp/audio.cpp-gguf/tree/main).

3. **Pick a preset** — the example audio.cpp preset lives in `data/shared/inference_presets/examples/`. Copy it to your local directory:

```bash
cp data/shared/inference_presets/examples/vulkan_8gb_audio_cpp.yml \
   data/shared/inference_presets/local/
```

Select it as usual via the preset switcher or Web UI.

4. **Restart the daemon** — the server will start with the new preset.

### Example Presets

| Preset | Notes |
|--------|-------|
| `vulkan_8gb_audio_cpp.yml` | Vulkan backend, web UI enabled (`--ui --ui-management`), single loaded model |

### How It Works

When `GF_AUDIO_CPP_ENABLE=1`, the daemon calls `data/audio_cpp/update_source.sh` during startup. The script:
- Clones the audio.cpp repo if missing, then pulls the latest source
- Compares the current git commit against `last_build.txt` and skips the build when unchanged
- Builds the `audiocpp_cli` and `audiocpp_server` binaries with the Vulkan backend and copies them into `data/audio_cpp/build/bin/`

The preset is built from a Dockerfile at `data/audio_cpp/docker/Dockerfile` (Ubuntu 26.04 with libgomp and Mesa Vulkan drivers) that runs `/app/audiocpp_server` as entrypoint. The preset mounts the build output to `/app` and the `AUDIO_CPP` models directory to `/app/models`. The inference server port (8080) is shared.

See the [audio.cpp server example docs](https://github.com/0xShug0/audio.cpp/blob/main/examples/docker/server/EXAMPLE.md) for server options.

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
| `DEGRADED` | Unexpected response |

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

### Request/Response Dumping

The proxy can record request and response pairs to JSON files for debugging and inspection. Enable it by setting `GF_PROXY_DUMP_REQUESTS=1` in your `.env` file:

```env
GF_PROXY_DUMP_REQUESTS=1
```

When enabled, each request/response pair is dumped to `docker/fifo_proxy/dumps/<timestamp>.json` with the following structure:

```json
{
  "requestMethod": "POST",
  "requestPath": "/v1/chat/completions",
  "request": "...",
  "response": "..."
}
```

The dump is written when the response completes or when the client disconnects. Dumped files are gitignored.

---

## Performance Statistics

GlacierFlow can log inference performance metrics — fill TPS, gen TPS, draft acceptance rate, and total tokens — to per-preset log files and visualize them in the Web UI.

### Setup

1. **Enable logging** by setting `GF_LOG_STATS=1` in your `.env` file:

```env
GF_LOG_STATS=1
```

2. Start the daemon. The daemon will write stats to `data/stats/<YYMMDD>_<preset_name>_<hash>.log` files — one per day per preset.

### Usage

Open the Stats Viewer from the Web UI toolbar link. The viewer loads the log files and renders them as an interactive Chart.js chart. Clear all button resets the chart view.

### Caveats

If preset is using llama's router mode you won't be able to recognize which model was actually called. 

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

The pi.dev container starts automatically with the daemon. You can access it in two ways:

**Web Terminal (TTYD):** Open `https://localhost:7681` in your browser (or click "pi.dev httpd" from the Web UI toolbar). Log in with the credentials set via `GF_HTTP_USER` / `GF_HTTP_PASSWORD` (defaults: `glacierflow` / `glacierflow`). The terminal launches inside a persistent tmux session. Note: if using the default self-signed certificate, you'll need to accept the browser's security warning.

**Management script:** For start/stop/attach control:

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

- **Image**: Alpine-based with bash, curl, ttyd, tmux, git, npm, and vim as basic requirements. Additional packages can be passed via `GF_PI_EXTRA_PACKAGES` .env arg (space-separated list).
- **User**: Runs as non-root (`pi_dev`, UID configurable via `GF_USER_ID`, defaults to 1000)
- **TTYD**: Web terminal server on port 7681 with basic auth (configurable via `GF_PI_DEV_TTYD_USER` / `GF_PI_DEV_TTYD_PASSWORD`)
- **Auto-update**: On startup, the container checks for and applies pi.dev updates automatically. Extensions are not pre-installed but can be added manually — they survive image rebuilds.
- **tmux**: Persistent session (`pi_dev_task`) with custom config mounted from `data/pi_dev/tmux.conf`
- **Volume mounts**:
  - Project root → `/pi_dev` (read-only)
  - Config → `~/.pi` (read-write)
  - Scripts → `/pi_dev_scripts` (read-write, for builtin automation scripts)
  - tmux config → `~/.tmux.conf` (read-only)
  - Workdir → configurable via `GF_PI_DEV_WORKDIR` env var (read-write)

---

## Hermes: AI Agent Container

GlacierFlow ships with a Dockerized Hermes agent environment for AI-driven task management, research, and kanban workflows. It runs inside an Ubuntu-based container with Hermes CLI, web dashboard, and gateway pre-installed.

### Setup

1. **Auto-start with daemon** — set `GF_HERMES_AUTOSTART=1` in your `.env` file to start the Hermes container automatically when the daemon launches:

```env
GF_HERMES_AUTOSTART=1
```

2. **Manual start** — use the management script:

```bash
cd scripts
./glacierflow_hermes
```

### Usage

The Hermes container provides multiple access points:

**Web Terminal (TTYD):** Open `https://localhost:7682` in your browser (or click "hermes httpd" from the Web UI toolbar). Log in with the credentials set via `GF_HTTP_USER` / `GF_HTTP_PASSWORD`.

**Hermes Dashboard:** Open `https://localhost:7683` in your browser (or click "hermes Web UI" from the Web UI toolbar) for the web-based Hermes dashboard.

### Configuring Hermes for Local Inference

To use your local llama.cpp inference server with Hermes:

1. Attach to the container or use the TTYD terminal
2. Run the configuration menu (`C` key) or `hermes setup`
3. Use `http://glacierflow-proxy:8080/v1` as the custom endpoint URL to route through the FIFO proxy

### Container Details

- **Image**: Ubuntu-based with bash, curl, ttyd, tmux, git, vim, python3, nodejs, npm, ripgrep, ffmpeg
- **User**: Runs as non-root (`ubuntu`, UID configurable via `GF_USER_ID` / `GF_USER_GROUP_ID`, defaults to 1000)
- **TTYD**: Web terminal server on port 7682 with basic auth
- **Dashboard**: Web UI on port 7683
- **tmux**: Persistent session (`hermes_task`)
- **Volume mounts**:
  - Hermes data → `data/hermes/` (read-write, configurable via `GF_HERMES_WORKDIR`)

---

## VS Code: Online Code Editor

GlacierFlow includes a [code-server](https://github.com/codercom/code-server) instance for web-based code editing. It shares the same workspace as pi.dev and is accessible through the Caddy reverse proxy.

### Setup

1. **Auto-start with daemon** — set `GF_VSCODE_AUTOSTART=1` in your `.env` file to start the VS Code container automatically when the daemon launches:

```env
GF_VSCODE_AUTOSTART=1
```

2. **Configure the version** (optional) — by default the latest image is used. Pin a specific version with:

```env
GF_VSCODE_VERSION=4.96.1
```

3. **Configure the port** (defaults to 7684):

```env
GF_VSCODE_WEB_PORT=7684
```

### Usage

Access VS Code at `https://localhost:7684` in your browser, or click "VS Code" from the Web UI toolbar.

### Container Details

- **Image**: `codercom/code-server` (latest by default, configurable via `GF_VSCODE_VERSION`)
- **User**: PUID/PGID configurable via `GF_USER_ID` / `GF_USER_GROUP_ID`
- **Healthcheck**: Monitors HTTP availability on port 8080
- **Volume mounts**:
  - Local data → `data/vscode/local/` (user settings, extensions)
  - Config → `data/vscode/config/` (code-server config)
  - Workdir → shared with pi.dev via `GF_PI_DEV_WORKDIR`
- **Caddy**: Reverse proxied on port 7684 through the Caddy service

---

## Remotes

- [GitHub](https://github.com/howanski/GlacierFlow)
- [Codeberg](https://codeberg.org/howanski/GlacierFlow)
- [Gitea (intranet)](https://gitea.howan.ski/howanski/GlacierFlow)

---

## Roadmap

- [x] **coder** - Code generation assistant integration
- [x] **coder automation** - Interactive menu, SKETCH→TODO pipeline, auto-programmer, auto-critic
- [x] **hermes** - Hermes AI agent container with dashboard and kanban support
- [x] **webui** - Web-based UI for preset management
- [x] **embeddings** - llama.cpp embedding server
- [x] **fifo-proxy** - Go-based FIFO proxy for serializing chat requests
- [x] **stats-viewer** - Web UI with Chart.js-based performance metrics visualization
- [x] **caddy-proxy** - Caddy reverse proxy with SSL/TLS termination and basic auth
- [x] **vscode** - Web-based code editor (code-server) shared workspace with pi.dev
- [x] **stable-diffusion** - Image generation server with automatic binary download/update
- [x] **audio.cpp** - Audio model inference server with automatic source clone/build

---

## License

GlacierFlow itself is a collection of custom scripts — no license applies.
Third-party software (llama.cpp, stable-diffusion.cpp, audio.cpp, Docker, pi.dev, llama-benchy, vscode and so on.) is subject to their respective licenses.
