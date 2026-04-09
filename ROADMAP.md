# syncctl Roadmap

## Planned

### Non-interactive CLI Enhancements
- `syncctl status --json` for machine-readable output
- `syncctl diff --name X` to preview changes before syncing

### Systemd Timer Integration
- `syncctl generate-systemd` outputs service + timer unit files
- Configurable interval (default: 30 minutes)
- Runs `syncctl sync --all` in headless mode

### Auto-sync Mode
- Watch for file changes via fsnotify
- Trigger sync after configurable debounce period (default: 30s)
- `syncctl watch` command or toggle in TUI

### Profile Support
- Named config profiles for different machines/contexts
- `syncctl --profile work` to load alternate config
- Useful for laptop vs. desktop setups

### Encryption Integration
- Support rclone crypt remotes
- Transparent encrypted sync with config guidance

### Parallel Diff Preview
- Before syncing, show what would change
- Fetch diffs for all mappings in parallel
- Color-coded additions/deletions/modifications

### Sync Scheduling via TUI
- Configure cron-like schedules per mapping directly in the TUI
- Generate and manage systemd timers from the dashboard

---

## Release and Distribution

### Build and Release
- GoReleaser config for cross-platform builds
- Version embedding via ldflags
- GitHub Releases automation

### Arch Linux (AUR)
- PKGBUILD for AUR distribution

### Dotfiles Integration
- Binary at `~/.local/bin/syncctl`
- Config symlinked from dotfiles repo
- Bootstrap: `go install github.com/CorpDK/bisync-tui/cmd/syncctl@latest`
