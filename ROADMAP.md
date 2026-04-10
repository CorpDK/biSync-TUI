# syncctl Roadmap

## Tier 1 — Must-Have (Reliability & Core)

### Reliability

- **Retry logic with exponential backoff**
  - Per-mapping `RetryPolicy`: max attempts, initial delay, backoff multiplier, max delay
  - Classify retryable vs non-retryable errors using rclone exit codes
  - Retry attempts visible in TUI output and recorded in history
  - Sane defaults: 3 attempts, 5s initial, 2x multiplier, 60s max

- **Post-sync verification**
  - Run `rclone check` (both directions) after successful sync to verify file integrity
  - Per-mapping toggle: `verify_after_sync = true/false`
  - Surface verification results in TUI detail panel and history
  - Support verification through crypt remotes

- **Graceful shutdown and interruption handling**
  - Add `StatusInterrupted` state distinct from `StatusError`
  - Detect `context.Canceled` in job execution and set appropriate status
  - Drain output channels properly on shutdown
  - Add shutdown timeout to prevent indefinite hangs

### Sync Hooks

- **Pre/post sync hooks**
  - Per-mapping config: `pre_sync_hook`, `post_sync_hook` (path to script)
  - Environment variables passed to hooks: `SYNCCTL_MAPPING_NAME`, `SYNCCTL_LOCAL`, `SYNCCTL_REMOTE`, `SYNCCTL_STATUS`, `SYNCCTL_DURATION`, `SYNCCTL_DRY_RUN`
  - Configurable pre-hook failure behavior: `pre_hook_fail_action = "abort" | "warn"`
  - Hook output captured in mapping logs
  - Timeout for hooks to prevent hangs

### Backup

- **Backup restore workflow**
  - `syncctl restore --name <mapping> --date <YYYY-MM-DD> [--to <path>]`
  - TUI action: "Restore from backup" — browse dates, preview contents, confirm
  - Selective file restore (individual files from a backup snapshot)
  - `RestoreFromBackup()` and `ListBackupContents()` methods on BackupManager

### Logging

- **Log rotation**
  - Config: `log_max_size_mb` (default 10), `log_max_files` (default 5)
  - Rotate when log exceeds max size (`.log.1`, `.log.2`, etc.)
  - Prevent unbounded disk usage from verbose rclone output

---

## Tier 2 — High Value (Differentiators)

### Scheduling & Automation

- **Systemd timer integration**
  - `syncctl generate-systemd` outputs service + timer unit files
  - Configurable interval (default: 30 minutes)
  - Runs `syncctl sync --all` in headless mode

- **Auto-sync mode (file watcher)**
  - Watch for file changes via fsnotify
  - Trigger sync after configurable debounce period (default: 30s)
  - `syncctl watch` command or toggle in TUI

- **Sync scheduling via TUI**
  - Configure cron-like schedules per mapping directly in the TUI
  - Generate and manage systemd timers from the dashboard

### Notifications

- **Pluggable notification backends**
  - Refactor to `Notifier` interface with multiple backends:
    - Desktop: `notify-send` (Linux), `osascript` (macOS), PowerShell toast (Windows)
    - Webhook: POST JSON to URL — covers Slack, Discord, ntfy.sh, Gotify, Pushover
    - Email: SMTP-based (optional, lower priority)
  - Configurable webhook templates with standard payload fields
  - Per-backend enable/disable and event toggles (success/failure)

### Organization

- **Mapping groups / tags**
  - Per-mapping: `tags = ["work", "personal", "critical"]`
  - `syncctl sync --tag <tag>` to sync subsets
  - TUI: filter mapping list by tag, "Sync tagged" action
  - Display tags in mapping list and detail panel

### Configuration

- **Config import/export**
  - `syncctl config export > my-config.toml`
  - `syncctl config import <file> [--merge | --replace]`
  - Validation on import before applying
  - Warn about sensitive data in exports

- **Bandwidth scheduling (time-of-day throttling)**
  - Support rclone's time-based bwlimit format: `"08:00,512k 18:00,off"`
  - TUI form for configuring bandwidth schedules per mapping

- **Exclude patterns UI / inline filters**
  - Inline config field: `exclude_patterns = ["node_modules/", ".git/", "*.tmp"]`
  - TUI action "Edit filters" with editable text area
  - Quick-add common patterns (`.git/`, `node_modules/`, `__pycache__/`, `*.tmp`)
  - Auto-generate filters file from inline patterns when `filters_file` is not set

### Observability

- **Audit trail**
  - Extend `HistoryRecord` with: trigger source (manual-tui/manual-cli/scheduled/auto-watch), sync options used, config hash
  - Separate append-only audit log: `$XDG_STATE_HOME/rclone-bisync/audit.jsonl`
  - Track config changes (mapping added/removed/modified)

---

## Tier 3 — Nice-to-Have (Polish & Advanced)

### Sync

- **Interactive conflict resolution**
  - Wire up the existing `ConflictsDetectedMsg` handler (currently a no-op)
  - Conflict resolution overlay showing path, Path1/Path2 info
  - Per-file choices: keep path1, keep path2, keep newer, keep both (rename), skip

- **Partial sync progress tracking**
  - Parse rclone `--progress` output for per-file progress
  - Persist progress in state: "Interrupted at 45/120 files"
  - Show resume context on next sync

- **Network-aware sync**
  - Config: `network_policy = "any" | "wifi-only" | "unmetered-only"`
  - Detect metered connections via NetworkManager D-Bus / `nmcli`
  - Skip sync with clear message when policy is not met

### Monitoring

- **Dashboard statistics view**
  - Aggregate stats: total syncs (today/week/month), data transferred, success rate, average duration
  - Most active and most error-prone mappings
  - Optional sparkline charts via charmbracelet libraries

- **Prometheus metrics export**
  - `syncctl metrics` HTTP server on configurable port
  - Metrics: `syncctl_sync_total`, `syncctl_sync_duration_seconds`, `syncctl_sync_bytes_transferred`, `syncctl_remote_healthy`, `syncctl_last_sync_timestamp`
  - Optional push-based via Pushgateway

- **Stale lock detection and visibility**
  - Show lock status per mapping in TUI (locked by PID)
  - `syncctl lock --clean` to remove stale lock files
  - Store PID in lock file content

### Scheduling

- **Cross-platform scheduling**
  - `syncctl generate-launchd` for macOS (LaunchAgent plist)
  - `syncctl generate-task-scheduler` for Windows (XML task definition)
  - Abstract `SchedulerBackend` interface: `Install()`, `Remove()`, `Status()`
  - Auto-detect platform and use appropriate backend

### UX

- **TUI help system**
  - Wire up the `?` key handler (currently a no-op)
  - Full keybinding help overlay with descriptions
  - First-run wizard when no config exists
  - Contextual hints in status bar based on current state

### Quality

- **Test infrastructure**
  - Unit tests for pure functions: `ParseDiffOutput`, `ParseConflicts`, `ParseProgress`, `ParseTransferSummary`, `StripANSI`, `classifyLevel`
  - `BuildBisyncArgs` tests (verify flag construction without rclone)
  - `HistoryStore`, `LockManager`, `LogManager` round-trip tests
  - Integration tests with mock rclone binary (shell script returning expected output)
  - Expand existing config validation tests

---

## Tier 4 — Future Considerations

- **Offline sync queue** — queue pending syncs when remote is down, auto-execute on reconnect
- **Config hot-reload** — auto-reload TOML when file changes externally while TUI is running
- **Mapping dependency chains** — ordered sync: mapping B only runs after mapping A succeeds
- **Snapshot / restic integration** — optional restic backend for deduplicated point-in-time backups
- **Proxy config surfacing** — show proxy env vars in TUI info panel, document proxy support
- **Deduplication awareness** — surface rclone `--dedup-mode` info for remotes that support it

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
