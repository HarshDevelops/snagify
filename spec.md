# Snagify Master Specification

## Project Overview

**Snagify** is a lightweight, single-binary CLI tool that diagnoses why a project "works on my machine" but fails on another developer's setup or in CI. 

It captures key environment and project configuration snapshots and produces clear, ranked diffs highlighting blockers.

**Tagline**: "Snag the differences. Kill 'works on my machine'."

**Core Value**: Fast mechanical diagnosis — not full reproducibility (Nix/Devbox territory), not pretty system info. Focus on actionable mismatches between two machines for a specific project.

**Target Users**: Developers, engineering teams, OSS maintainers, DevOps engineers who onboard teammates or debug cross-environment issues.

**Differentiation**:
- Narrow, high-signal capture focused on dev workflows.
- Ranked output: Critical blockers first.
- Zero AI, zero bloat, instant 10-30s workflow.
- Cross-platform (Linux, macOS, Windows).

**Success Metric for MVP**: A dev runs `snagify snapshot` on two machines and immediately sees the exact reason for setup failure. Demo should elicit "I need this" response.

## Goals

- **MVP in 7-10 days**: Deliver core `snapshot` + `diff` with excellent UX.
- Production-grade quality from day 1: Robust error handling, clear output, good defaults.
- Single static binary, easy install (`go install` or curl).
- Excellent documentation and demo in README.`
- No external runtime dependencies beyond standard tools (Docker, Node, etc. for detection).

## Non-Goals (v0.1)

- Auto-fixes or `doctor` command.
- Full OS package scanning.
- Deep Docker/Podman container inspection.
- Secret scanning or .env value comparison (only presence/missing keys).
- TUI / interactive mode.
- GitHub Actions / CI integration.
- Config files (hardcoded sensible defaults).
- Lockfile deep parsing.
- GUI.

## Tech Stack

- **Language**: Go (1.23+ preferred) — for speed, single binary, cross-compilation.
- **CLI Framework**: `github.com/spf13/cobra`
- **Output Styling**: `github.com/charmbracelet/lipgloss` + `github.com/charmbracelet/table` for beautiful terminal tables.
- **JSON Handling**: Standard `encoding/json`.
- **Version Detection**: `os/exec` + parsing common `--version` outputs.
- **No heavy deps**: Keep `go.mod` minimal.

## Commands (v0.1)

### 1. `snagify snapshot [output.json]`
- Default output: `./snagify-snapshot-$(hostname)-$(date).json`
- Captures current machine/project state.
- Detects project root (look for common manifest files upward from cwd).
- Quiet mode with `--quiet` / `-q`.

### 2. `snagify diff <snapshotA.json> <snapshotB.json>`
- The core command.
- Outputs human-readable ranked diffs to stdout.
- Supports `--format markdown` for GitHub issues.
- Supports `--format json` for scripting.
- Exit code: 0 = no differences, 1 = differences found.

**Global Flags**:
- `--project-root string` : Override auto-detection.
- `--verbose` / `-v`

## Snapshot Data Model

Define a Go struct `Snapshot`:

```go
type Snapshot struct {
    Timestamp   string                 `json:"timestamp"`
    Hostname    string                 `json:"hostname"`
    Project     ProjectInfo            `json:"project"`
    Environment Environment            `json:"environment"`
    Runtimes    Runtimes               `json:"runtimes"`
    Services    Services               `json:"services"`
    EnvFiles    EnvFiles               `json:"env_files"`
}

type ProjectInfo struct {
    Root          string   `json:"root"`
    Manifests     []string `json:"manifests"` // e.g. package.json, go.mod
    Name          string   `json:"name,omitempty"`
}

type Environment struct {
    OS            string `json:"os"`
    Arch          string `json:"arch"`
    OSVersion     string `json:"os_version,omitempty"`
    GitVersion    string `json:"git_version"`
}

type Runtimes struct {
    Node      VersionInfo `json:"node"`
    NPM       VersionInfo `json:"npm"`
    PNPM      VersionInfo `json:"pnpm"`
    Yarn      VersionInfo `json:"yarn"`
    Python    VersionInfo `json:"python"`
    Pip       VersionInfo `json:"pip"`
    UV        VersionInfo `json:"uv"`
    Java      VersionInfo `json:"java"`
    Maven     VersionInfo `json:"maven"`
    Gradle    VersionInfo `json:"gradle"`
    Go        VersionInfo `json:"go"`
    Rust      VersionInfo `json:"rust"`
    Cargo     VersionInfo `json:"cargo"`
    Docker    VersionInfo `json:"docker"`
    // Podman support later
}

type VersionInfo struct {
    Version string `json:"version"`
    Present bool   `json:"present"`
    Path    string `json:"path,omitempty"`
}

type Services struct {
    Ports map[int]PortInfo `json:"ports"`
}

type PortInfo struct {
    Port     int    `json:"port"`
    Service  string `json:"service,omitempty"`
    Listening bool   `json:"listening"`
}

type EnvFiles struct {
    EnvExampleExists bool     `json:"env_example_exists"`
    EnvExists        bool     `json:"env_exists"`
    MissingKeys      []string `json:"missing_keys"`
}
```

## Capture Logic (Detailed)

Implement helper functions for each category. Use `exec.Command` with timeout.

**Project Detection**:
- Walk up from cwd looking for: `package.json`, `go.mod`, `pom.xml`, `build.gradle`, `requirements.txt`, `Cargo.toml`, etc.
- Extract name if possible (e.g. from package.json).

**Runtimes Detection** (examples):
- Node: `node --version`
- npm/pnpm/yarn: respective `--version`
- Python: `python --version` and `python3 --version`
- Java: `java -version`
- Maven/Gradle: `--version`
- Go: `go version`
- Rust: `rustc --version`
- Cargo: `cargo --version`
- Docker: `docker --version`
- Handle "command not found" gracefully (Present: false).

**Ports**:
- Common dev ports: 3000 (dev server), 5432 (Postgres), 6379 (Redis), 8080, 9200 (ES), 27017 (Mongo), etc.
- Use cross-platform: On Unix `ss -tlnp` or `lsof`, on Windows `netstat`. Fallback to simple checks.

**Env Files**:
- Look for `.env.example` and `.env` in project root.
- Parse keys from `.env.example` (simple line split on `=`).
- Report missing keys in `.env`.

**Error Handling**:
- Continue on partial failures (e.g. one runtime missing).
- Log warnings to stderr if verbose.

## Diff Logic & Ranking

**Core Algorithm**:
1. Load both snapshots.
2. Compare field-by-field.
3. Categorize differences:

**Categories**:
- **Critical / Likely Blockers**:
  - Missing required runtime (Docker, Java+ Maven for Java projects, etc.)
  - Major version mismatch (e.g. Node 18 vs 22)
  - Missing required .env keys
  - Required port not listening (inferred from project type)

- **Differences (potentially ok)**:
  - Minor version differences
  - Optional runtime differences

- **Informational**:
  - Extra ports, OS details, etc.

**Output Structure**:
- Project header
- Ranked sections with tables
- Summary "Likely blockers" paragraph with plain English explanation.
- Use colors (lipgloss): Red for critical, Yellow for differences, Gray for info.

**Markdown Format**:
- Clean GitHub-friendly tables and code blocks.

## User Experience & Output Examples

**Snapshot Command**:
```
$ snagify snapshot
Snapshot saved to snagify-snapshot-myhost-20260606.json
```

**Diff Command** (Killer Demo):
```
$ snagify diff harsh.json teammate.json

Project: checkout-service
Critical differences
────────────────────
Java        harsh: 17.0.9      teammate: 21.0.2
Maven       harsh: 3.9.6       teammate: missing          ← Likely blocker
Docker      harsh: 25.0        teammate: not installed
Postgres    harsh: running:5432 teammate: not running
.env        teammate missing: DATABASE_URL, JWT_SECRET

Likely blockers for teammate:
• Maven is missing
• Docker is not installed
• Postgres is not running
• Required environment variables are absent

Run `snagify snapshot` on teammate's machine to capture latest.
```

## Installation & Distribution

- `go install github.com/yourusername/snagify@latest`
- GitHub Releases with binaries for all platforms.
- One-liner install script (optional).

## Testing Strategy

- Unit tests for parsers and diff logic.
- Integration tests with mocked exec outputs.
- Manual testing across Linux/macOS/Windows VMs.
- Test cases: Identical machines, missing runtimes, port conflicts, .env mismatches.

## Error Cases & Edge Cases

- No project detected
- Command not found for runtimes
- Permission issues (ports)
- Malformed JSON snapshots
- Windows path handling
- Very large outputs (unlikely)

## Future Roadmap (Post-MVP)

- v0.2: Config file `.snagify.yaml`, custom runtimes/ports, `doctor` suggestions.
- v0.3: GitHub Action, basic auto-fixes.
- v1.0: Plugin system for project-specific checks.

## Development Setup

1. `git clone ...`
2. `go mod tidy`
3. `go run main.go snapshot`
4. Build: `go build -o snagify`

## README.md Requirements (Separate)

- Hero demo GIF/terminal recording (critical for stars).
- 30-second quickstart.
- Full examples.
- Comparison to alternatives.
- Contributing guide.

## License

MIT

---

**End of Spec** — This document is complete enough for an AI coding assistant to implement a production-grade v0.1. Prioritize clean code, excellent output formatting, and robustness.
