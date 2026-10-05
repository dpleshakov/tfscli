# Changelog

## [Unreleased]

### Added
### Fixed
- A collection, project, or team name containing `%`, such as `100% Done`, now reaches the server as written, where the request used to go to the wrong address.
### Changed
### Removed

---

## [0.0.6] — 2026-10-04

### Added
- `tfscli wit work-items list` prints several work items by id with one request, taking the ids in `--ids` and the optional `--fields`, `--as-of`, and `--error-policy`.
- `tfscli wit work-items get-batch` does the same with a POST request, for lists of ids too long for the URL of `wit work-items list`, on Azure DevOps Server 2019 or later.
- `--error-policy omit` on both commands prints the work items that exist and lists each id the server did not return, instead of failing the command.
- `tfscli wit wiql query-by-wiql` runs the WIQL query given in `--query` and prints the ids of the work items found, or the links between them for a tree or one-hop query, with the optional `-p`, `--team`, `--top`, and `--time-precision`; without a project the query runs across the collection.

### Fixed
- A mistyped command name after the area, such as `tfscli wit workitems get`, is now reported as a `config` error that suggests the closest command, instead of printing help and exiting with 0.

### Changed
- `tfscli workitem get` is now `tfscli wit work-items get`: commands are named by the REST API area, the resource, and the operation, as the paths of the REST API reference name them, and the old name is no longer accepted.

---

## [0.0.5] — 2026-10-04

### Fixed
- `tfscli auth login` now succeeds on servers that accept a personal access token only within its collection, where it used to fail with an `auth` error for a valid token.

### Changed
- The collection is now part of the credential: `tfscli auth login` asks for it between the server URL and the token, `auth.json` and `TFSCLI_AUTH` require a `collection` field, and the token is sent only to the URL and the collection stored with it.
- `tfscli auth login` now removes the collection from the end of the server URL when the URL was entered with it, and says which server URL it uses instead.
- `tfscli auth login` now prints the stored collection together with the server URL and the user name.
- A server URL or collection that leads nowhere during `tfscli auth login` is now reported with the address they made up and a request to check both, instead of `resource not found`.
- An HTTP 401 is now reported as the server not accepting the token, followed by its possible causes, including IIS Basic Authentication on the server, instead of `PAT is invalid or expired`.

### Removed
- `--collection`, `TFSCLI_COLLECTION`, and the `collection` key of the config file no longer exist; the collection comes from `tfscli auth login` or `TFSCLI_AUTH`.

---

## [0.0.4] — 2026-10-04

### Changed
- Requests no longer carry `api-version=7.2` by default: without `--api-version`, `TFSCLI_API_VERSION`, or `apiVersion`, no version is sent and the server answers at the version it chooses, so servers that do not support REST API 7.2 work without configuration.
- `tfscli auth login` now checks the credential without an API version: `apiVersion` from the config file no longer applies to it, and `--api-version` on it is refused with a `config` error.
- An API version refused by the server is now reported together with the setting it came from — `--api-version`, `TFSCLI_API_VERSION`, or `apiVersion` in the config file — and how to fix it.

---

## [0.0.3] — 2026-10-01

### Added
- `tfscli auth login` asks for the server URL and a personal access token with hidden input, checks them against the server, and stores them together in `$XDG_DATA_HOME/tfscli/auth.json` (`~/.local/share/tfscli/auth.json` by default), readable by the current user only.
- `TFSCLI_AUTH` supplies the same credential as JSON, `{"url": "…", "pat": "…"}`, where an interactive login is not possible, such as in CI.

### Changed
- The config file is now read from `$XDG_CONFIG_HOME/tfscli/config.json`, or `~/.config/tfscli/config.json` when `XDG_CONFIG_HOME` is not set, on every OS including Windows; `~/.tfscli/config.json` is no longer read.
- The personal access token is now sent only to the server URL stored with it; the server URL and the token always come from the same source, `auth.json` or `TFSCLI_AUTH`.

### Removed
- `--url`, `--pat`, `TFSCLI_URL`, `TFSCLI_PAT`, and the `url` and `pat` keys of the config file no longer exist; `tfscli auth login` or `TFSCLI_AUTH` replaces them.

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
