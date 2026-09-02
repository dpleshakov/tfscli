# Changelog

## [Unreleased]

### Added
### Fixed
### Changed
### Removed

---

## [0.0.2] — 2026-09-02

### Added
- Every release archive now carries an agent skill in `skills/tfscli/SKILL.md`, which teaches an AI agent — Claude Code or opencode — when and how to call tfscli.

---

## [0.0.1] — 2026-08-31

### Added
- `tfscli workitem get -p <project> <id>` prints a single work item as markdown, converting HTML fields such as Description and Repro Steps.
- `--fields` limits the request to the named fields and prints them in the order they were asked for.
- Configuration is read from `~/.tfscli/config.json`, `TFSCLI_*` environment variables, and command-line flags, in that order of precedence.
- `caBundle` and `insecureSkipVerify` in the config file allow connecting to a server whose certificate is issued by an internal CA.
- `--verbose` (or `TFSCLI_VERBOSE=1`) logs every request to stderr with its method, URL, status, and duration.
- Errors are reported on stderr as `Error [category]: message (HTTP status)` with a non-zero exit code, using the categories `auth`, `not_found`, `forbidden`, `server`, `config`, and `network`.
- `--version` prints the version and the commit the binary was built from.
