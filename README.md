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
- Docker-based code generation assistant (pi.dev) with automation workflows
- Multi-model benchmarking suite


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
│   ├── llama_cpp/
│   │   ├── compose.yml              # Base Docker Compose template
│   │   └── compose.override.yml     # Generated from preset (gitignored)
│   └── pi_dev/                      # Code generation assistant (coder)
│       ├── Dockerfile               # pi.dev dev environment image
│       ├── compose.yml              # Container orchestration
│       ├── entrypoint               # Health tick script
│       └── start                    # Interactive kick-off & automation menu
├── data/
│   ├── models/                      # Model storage (gitkeep placeholder)
│   ├── shared/
│   │   ├── inference_config.yml     # Active config source (gitignored)
│   │   ├── inference_presets/
│   │   │   ├── examples/            # Example presets to copy
│   │   │   │   ├── vulkan_8gb_gemma4_e4b_q5.yml
│   │   │   │   └── vulkan_8gb_qwen36_35b_q6_coding.yml
│   │   │   └── local/               # Your custom presets (gitignored)
│   │   ├── benchmarks/              # Benchmark test cases & results
│   │   │   ├── c1.json              # SQL query generation
│   │   │   ├── c2.json              # Bash script generation
│   │   │   ├── c3.json              # PHP optimization
│   │   │   ├── p1.json              # Creative writing
│   │   │   └── *.txt                # Benchmark results (gitignored)
│   │   ├── inference_status.txt     # Current status (gitignored)
│   │   ├── inference_preset_name.txt      # Currently active preset (gitignored)
│   │   ├── inference_preset_name_target.txt # Desired preset (gitignored)
│   │   ├── inference_loading_times.txt  # Load time history (gitignored)
│   │   ├── inference_up_hash.txt      # Hash of running model (gitignored)
│   │   ├── inference_up_timestamp.txt # Start timestamp (gitignored)
│   │   └── INFERENCE_LOCK           # Lock file (gitignored)
│   └── pi_dev/
│       ├── config/
│       │   └── agent/
│       │       ├── auth.json        # Authentication config (gitignored)
│       │       ├── settings.json    # Agent settings (gitignored)
│       │       ├── models.json      # Model provider configuration
│       │       ├── bin/             # Bundled binaries (gitignored)
│       │       │   ├── fd           # Fast file finder
│       │       │   └── rg           # Fast grep (ripgrep)
│       │       └── sessions/        # Session history (gitignored)
│       └── scripts/
│           └── builtin/             # Automation scripts
│               ├── auto_critic.txt
│               ├── auto_planner.txt
│               ├── auto_programmer.txt
│               ├── init_code_review.txt
│               ├── init_improvements.txt
│               └── init_readme.txt
├── scripts/
│   ├── gf_common.sh                 # Shared library (constants & utils)
│   ├── glacierflow_host_daemon      # Background service
│   ├── glacierflow_inference_select_preset  # Preset selector CLI
│   ├── glacierflow_benchmark        # Multi-model benchmark runner
│   └── glacierflow_pi_code          # pi.dev container management
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

### 3. Configure environment

Create a `.env` file with your paths:

```env
GF_LLAMA_MEMORY_LIMIT=50g
GF_LLAMA_SERVER_VERSION=server-vulkan
GF_MODELS_DIRECTORY=/path/to/your/models
GF_PI_DEV_WORKDIR=/path/to/coder/workdir
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

## Coder: Code Generation Assistant

GlacierFlow ships with a Dockerized [pi.dev](https://pi.dev) development environment for AI-assisted code generation. It runs inside a full-featured dev container with all common languages and tools pre-installed.

### Setup

1. **Configure the model provider** in `data/pi_dev/config/agent/models.json`:

```json
{
  "providers": {
    "glacierflow-llamacpp": {
      "baseUrl": "http://glacierflow-llamacpp:8080/v1",
      "api": "openai-completions",
      "apiKey": "none",
      "models": [
        { "id": "glacierflow-llamacpp" }
      ]
    }
  }
}
```

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
- [ ] **webui** - Web-based UI for preset management
- [ ] **agent** - Autonomous agent mode

---

## License

GlacierFlow itself is a collection of custom scripts — no license applies.
Third-party software (llama.cpp, Docker, pi.dev etc.) is subject to their respective licenses.
