# chart_version_updater

A CLI tool that automatically updates Helm chart versions in Argo CD Application manifests by fetching the latest stable versions from [ArtifactHub](https://artifacthub.io/).

## Overview

This tool solves the problem of keeping Helm chart versions up-to-date across multiple Argo CD Applications. Instead of manually checking for new versions and editing YAML files, you add a simple comment to your manifests and run this tool to automatically update them.

### How It Works

1. Scans a directory for Argo CD Application manifests (`.yaml`/`.yml` files)
2. Looks for manifests with an `# artifacthub:` comment specifying the ArtifactHub repository
3. For each chart, fetches available versions from the ArtifactHub API
4. Filters out pre-release versions (those containing `-`)
5. Compares the current `spec.source.targetRevision` with the latest stable version
6. Updates the YAML file if a newer version is available

## Installation

```bash
# Clone and build
git clone <repo-url>
cd chart_version_updater
make build
```

Requires Go 1.25.6 or later.

## Usage

```bash
# Update all charts to latest versions
./update-version

# Specify a custom directory
./update-version --dir ./my-apps

# Preview changes without modifying files (shows git diff)
./update-version --dry-run

# Discover charts and show what would be updated
./update-version --check
```

### Command-Line Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--dir <path>` | `-d` | Path to directory containing Argo CD Application manifests (default: `argoapps`) |
| `--dry-run` | `-n` | Show git diff without modifying files |
| `--check` | `-C` | Discover charts and show what would be updated |
| `--help` | `-h` | Show help message |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `UPDATE_VERSION_DIR` | Directory path (used if `--dir` is not provided) |

## Configuration

No separate configuration file is needed. Simply add an `# artifacthub:` comment at the top of your Argo CD Application manifests:

```yaml
# artifacthub: cilium/cilium
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cilium
  namespace: argocd
spec:
  source:
    repoURL: https://helm.cilium.io
    chart: cilium
    targetRevision: 1.16.0  # <-- This field gets updated
  destination:
    server: https://kubernetes.default.svc
    namespace: kube-system
```

### Comment Format

The comment must be:
- At the very top of the file (before the `apiVersion` line)
- In the format `# artifacthub: <org>/<repo>`
- The `<org>/<repo>` corresponds to the ArtifactHub package path

### Finding ArtifactHub Repository Paths

To find the correct repository path for a chart:

1. Go to [artifacthub.io](https://artifacthub.io/)
2. Search for your chart
3. The URL will be `https://artifacthub.io/packages/helm/<org>/<repo>`
4. Use `<org>/<repo>` as your comment value

Examples:
- Cilium: `cilium/cilium`
- Argo CD: `argo/argo-cd`
- cert-manager: `cert-manager/cert-manager`
- Longhorn: `longhorn/longhorn`

### Multi-Document YAML Files

For files containing multiple YAML documents (separated by `---`), the tool looks for the `Application` kind and updates its `targetRevision`. Other documents in the file (like Secrets or NetworkPolicies) are preserved.

## Project Structure

```
.
├── main.go           # CLI entry point and argument parsing
├── config.go         # Directory scanning and chart discovery
├── update.go         # Chart update orchestration
├── artifacthub.go    # ArtifactHub API client
├── version.go        # Semantic version comparison
├── yaml.go           # YAML document reading/writing with AST preservation
├── diff.go           # Git diff display for dry-run mode
├── util.go           # Logging and error handling utilities
└── Makefile          # Build and development commands
```

## Development

```bash
make help  # Show all available targets
```

### Make Targets

| Target | Description |
|--------|-------------|
| `build` | Build binary for current platform |
| `build-all` | Build binaries for all platforms (darwin, linux, freebsd) |
| `build-darwin` | Build binaries for macOS (amd64, arm64) |
| `build-linux` | Build binaries for Linux (amd64, arm64) |
| `build-freebsd` | Build binaries for FreeBSD (amd64, arm64) |
| `clean` | Remove built binaries |
| `test` | Run tests |
| `lint` | Run golangci-lint |
| `fmt` | Format code with go fmt |
| `vet` | Run go vet |
| `run` | Build and run the binary |
| `check` | Build and run with -C flag (discover charts) |
| `help` | Show available targets |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (no charts found, network failure, file not found, etc.) |

## Security

- Path traversal protection: Only files within the specified directory are processed
- HTTP timeout: 60-second timeout on ArtifactHub API requests
- Pre-release filtering: Versions containing `-` are automatically excluded

## Dependencies

- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing with AST preservation

## License

[GNU General Public License v3.0 only](https://spdx.org/licenses/GPL-3.0-only.html)
