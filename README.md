```
   ____ _            _           _____ _               
  / ___| | __ _  ___(_) ___ _ __|  ___| | _____      __
 | |  _| |/ _` |/ __| |/ _ \ '__| |_  | |/ _ \ \ /\ / /
 | |_| | | (_| | (__| |  __/ |  |  _| | | (_) \ V  V / 
  \____|_|\__,_|\___|_|\___|_|  |_|   |_|\___/ \_/\_/  
                                                       
```

# GlacierFlow

A lightweight toolkit for managing local AI inference servers (llama.cpp) on Linux, with hot-swap model presets and automatic container orchestration.

---

## What It Does

GlacierFlow runs llama.cpp inside Docker and lets you switch between model presets on the fly — no manual container restarts needed. A background daemon watches your config and applies changes automatically.

**Current features:**
- llama.cpp inference server managed via Docker Compose
- Hot-swap between inference presets (YAML configs)
- Automatic container restarts when preset changes
- Preset loading time tracking & status monitoring
- Example configs for various GPU setups

---

## Architecture

```
┌──────────────────────┐
│  glacierflow_host_daemon   ← Background watcher (runs every 10s)
│  - Monitors inference_config.yml   ← Detects preset changes
│  - Manages Docker Compose lifecycle  ← Starts/stops container
│  - Health-checks the server          ← Checks /ping endpoint
└──────────┬─────────────┘
           │ updates
           ▼
┌──────────────────────┐     ┌──────────────────────┐
│ glacierflow_inference│────▶│  data/shared/        │
│    _select_preset    │     │  inference_config.yml│
│                      │     └──────────┬───────────┘
│  List available      │                │ copies
│  Load by name/hash   │                ▼
│  Track load times    │     ┌──────────────────────┐
└──────────────────────┘     │ docker/llama_cpp/    │
                             │  compose.override.yml│
                             └──────────┬───────────┘
                                        │
                                        ▼
                             ┌──────────────────────┐
                             │  llama.cpp (Docker)  │
                             │  :8080 (HTTP API)    │
                             └──────────────────────┘
```

---

## Prerequisites

- Linux (tested on Arch/Manjaro)
- Docker & Docker Compose
- GPU drivers (Vulkan for AMD/NVIDIA)
- Bash, curl, grep, awk, md5sum, and other coreutils

---

## Project Structure

```
.
├── docker/
│   └── llama_cpp/
│       ├── compose.yml              # Base Docker Compose template
│       └── compose.override.yml     # Generated from preset (gitignored)
├── data/
│   └── shared/
│       ├── inference_config.yml     # Active config source (gitignored)
│       ├── inference_presets/
│       │   ├── examples/            # Example presets to copy
│       │   └── local/               # Your custom presets
│       ├── inference_status.txt     # Current status (gitignored)
│       ├── inference_preset_name.txt
│       ├── inference_loading_times.txt
│       └── INFERENCE_LOCK           # Lock file (gitignored)
├── scripts/
│   ├── glacierflow_host_daemon      # Background service
│   └── glacierflow_inference_select_preset  # Preset selector CLI
├── .env                             # Environment variables (gitignored)
└── README.md
```

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

Then copy it as the active config:

```bash
cp data/shared/inference_presets/local/vulkan_8gb_gemma4_e4b_q5.yml \
   data/shared/inference_config.yml
```

### 3. Configure environment

Create a `.env` file with your paths:

```env
GF_LLAMA_SERVER_VERSION=server-vulkan
GF_LLAMA_MEMORY_LIMIT=50g
GF_MODELS_DIRECTORY=/path/to/your/models
```

### 4. Start the daemon

Run the background service to spin up the inference server:

```bash
./scripts/glacierflow_host_daemon
```

For a one-shot check (no daemon loop):

```bash
./scripts/glacierflow_host_daemon --once
```

### 5. Switch presets

List available presets:

```bash
./scripts/glacierflow_inference_select_preset
```

Load a preset by name or hash:

```bash
./scripts/glacierflow_inference_select_preset vulkan_8gb_gemma4_e4b_q5
./scripts/glacierflow_inference_select_preset <md5hash>
```

---

## Preset Files

Preset files are Docker Compose YAML snippets that override the base `compose.yml`. They define:

- **command**: llama.cpp server arguments (model path, context size, GPU layers, etc.)
- **devices**: GPU device passthrough (e.g., `/dev/dri/renderD128` for AMD)
- **volumes**: Model directory mount paths


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

## Remotes

- [GitHub](https://github.com/howanski/GlacierFlow)
- [Codeberg](https://codeberg.org/howanski/GlacierFlow)
- [Gitea (intranet)](https://gitea.howan.ski/howanski/GlacierFlow)

---

## Roadmap

- [ ] **coder** — Code generation assistant integration
- [ ] **agent** — Autonomous agent mode
- [ ] **webui** — Web-based UI for preset management

---

## License

GlacierFlow itself is a collection of custom scripts — no license applies.
Third-party software (llama.cpp, Docker, etc.) is subject to their respective licenses.
