# ChangeLog

All notable changes to this project will be documented in this file.

## \[Unreleased\]

### Added

- Added option to set `--cache-freq`.
- Added option to set `--zap-log-level`.
- Added more data fields to existing collections -- expanded node and job
  states, memory usage.
- Added accounting data collection -- job states, TRES usage.

### Fixed

- Fixed image tag incorrectly defaulting to appVersion instead of version.

### Changed

- Changed `--server` default to localhost URL.

### Removed

- Removed `--per-user-metrics` toggle, enabled automatically.
