# 2026-10-01-tasks-xdg-paths.md

**Status:** Archived

## Context

The config file lives in `~/.tfscli/config.json`, a location of the project's
own invention. The work moves it to the XDG Base Directory layout, which
cross-platform developer tools such as opencode already follow, so that the
location is the one users expect. Nothing has been distributed to users yet,
so the move needs no migration; after a public release it would.

Decisions taken during planning, recorded so they are not re-litigated:

- XDG is applied on every OS, Windows included: `$XDG_CONFIG_HOME/tfscli/`,
  defaulting to `~/.config/tfscli/`. `%AppData%`, which `os.UserConfigDir`
  returns on Windows, is deliberately not used: one path across platforms keeps
  `README.md` and `SKILL.md` simple, developer tools on Windows already keep
  their files in the user profile, and `%AppData%\Roaming` is copied to a file
  server under roaming profiles.
- The old location is not read and no migration is provided.
- This work precedes `2026-10-01-tasks-auth-login.md`, which places the
  authentication file under `$XDG_DATA_HOME` by the same rules.

---

### TASK-01 `xdg-config-path`
**Description:** Resolve the config file path as
`$XDG_CONFIG_HOME/tfscli/config.json`, falling back to
`~/.config/tfscli/config.json` when the variable is unset or empty, on every
OS. Update the tests, the "config file not found at …" message, `README.md`,
`skills/tfscli/SKILL.md`, `docs/project-brief.md`, `docs/architecture.md`, and
`CLAUDE.md` wherever they name the path. Record the change in `CHANGELOG.md`
through the `changelog` skill, section `Changed`.
**Definition of done:** The config path is derived from XDG rules alone on
every OS, and tests cover both `XDG_CONFIG_HOME` set and unset. `~/.tfscli`
no longer appears in the repository outside `docs/archive/` and `CHANGELOG.md`.
`make check` passes.
**Status:** Done
