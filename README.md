# prunesh/pip

Token-reduction plugin for [prunesh](https://github.com/prunesh/prunesh) that filters `pip` output.

A typical `pip install -r requirements.txt` with 30 packages produces 200–500 lines of download progress,
dependency resolution, and wheel-building noise. This plugin compresses that to the lines that actually matter.

## What it filters

| Subcommand | Stripped | Kept |
|---|---|---|
| `install` / `download` | `Collecting`, `Downloading`, `Using cached`, progress bars, `Building wheel` | `Successfully installed`, `Installing collected packages`, `WARNING`, errors |
| `uninstall` | Verbose preamble | `Successfully uninstalled X` |
| `show` | Author, License, Home-page, Summary, … | Name, Version, Location, Requires, Required-by |
| `list` | — | First 50 packages + `... +N more` notice |
| `freeze` / `check` | — | Full passthrough |

### Example

**Before** (`pip install torch numpy`):
```
Collecting torch==2.0.0
  Downloading torch-2.0.0-cp311-cp311-manylinux1_x86_64.whl (619.9 MB)
     ━━━━━━━━━━━━━━━━━━━━━━━━━ 619.9/619.9 MB 8.2 MB/s eta 0:00:00
Collecting numpy>=1.21.0 (from torch==2.0.0)
  Using cached numpy-1.26.4-cp311-cp311-manylinux_2_17_x86_64.whl (17.3 MB)
Installing collected packages: numpy, torch
Successfully installed numpy-1.26.4 torch-2.0.0
```

**After**:
```
Installing collected packages: numpy, torch
Successfully installed numpy-1.26.4 torch-2.0.0
```

## Install

Requires [prunesh core](https://github.com/prunesh/prunesh) >= 0.12.0.

```bash
prunesh plugin install github.com/prunesh/pip@v0.1.0
```

To replace an existing `pip` plugin:

```bash
prunesh plugin install github.com/prunesh/pip@v0.1.0 --replace
```

## Uninstall

```bash
prunesh plugin uninstall prunesh/pip
```

## How it works

The plugin speaks the `stdin/v1` protocol with the prunesh core proxy:

- **Rewrite**: injects `--progress-bar=off` into `install`/`download`/`wheel` invocations when no quieting flag (`-q`, `--quiet`) is already present. This eliminates multi-line download-progress output before filtering even runs.
- **FilterOutput**: applies rule-based heuristics to strip noise and keep actionable lines.

## pip3

`pip3` is a separate argv0. A `prunesh/pip3` plugin with identical logic will be published separately.

## License

MIT
