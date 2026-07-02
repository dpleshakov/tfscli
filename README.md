# tfscli

A stateless CLI for read-only access to on-premises TFS / Azure DevOps Server, designed for AI coding agents and equally usable by humans. Single binary, PAT authentication, markdown output.

## Status

Pre-release. The repository is scaffolded; no functionality is implemented yet. See `docs/project-brief.md` for scope and `docs/architecture.md` for design decisions.

## Configuration

Copy `config.example.json` to `~/.tfscli/config.json` and fill in your server URL, collection, and PAT. Every value can be overridden by `TFSCLI_*` environment variables and command-line flags (flag > env > file > default).

## Build

```
go build ./...
```
