# k9sight

A fast, keyboard-driven TUI for debugging Kubernetes workloads.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

## Features

- Browse deployments, statefulsets, daemonsets, jobs, cronjobs
- View pod logs with search, time filtering, and container selection
- Execute into pods, port-forward, and describe directly from TUI
- Scale and restart workloads
- Monitor events and resource metrics
- Debug helpers for common issues (CrashLoopBackOff, ImagePullBackOff, etc.)
- Vim-style navigation

## Install

**Homebrew:**
```bash
brew install doganarif/tap/k9sight
```

**Go:**
```bash
go install github.com/doganarif/k9sight/cmd/k9sight@latest
```

**From source:**
```bash
git clone https://github.com/doganarif/k9sight.git
cd k9sight
go build -o k9sight ./cmd/k9sight
```

## Usage

```bash
k9sight
```

### Key Bindings

**Navigation**
| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `enter` | Select |
| `esc` | Back / Close |
| `/` | Search/Filter |
| `n` | Change namespace |
| `t` | Change resource type |
| `?` | Help |
| `q` | Quit |

**Workload Actions**
| Key | Action |
|-----|--------|
| `s` | Scale deployment/statefulset |
| `R` | Restart workload |

**Pod Actions** (in pod view)
| Key | Action |
|-----|--------|
| `a` | Actions menu (exec, port-forward, describe, delete) |
| `y` | Copy kubectl commands |

**Logs Panel**
| Key | Action |
|-----|--------|
| `/` | Search logs |
| `[` `]` | Cycle containers |
| `P` | Previous container logs |
| `T` | Time filter (5m/15m/1h/6h) |
| `f` | Toggle follow |
| `e` | Jump to next error |

**Panels**
| Key | Action |
|-----|--------|
| `1-4` | Focus panel (logs/events/metrics/manifest) |
| `tab` | Next panel |
| `v` | Fullscreen toggle |

## Requirements

- Go 1.21+
- kubectl configured with cluster access

## License

MIT
