# 2026-10-01-tasks-remove-pat-flag.md

**Status:** Archived

## Context

The global `--pat` flag puts the personal access token on the command line,
where it lands in shell history, in the process list, and in the transcript of
any agent session that runs the command. No supported scenario needs it: the
config file and `TFSCLI_PAT` cover both interactive and CI use.

Decisions taken during planning, recorded so they are not re-litigated:

- The PAT is supplied only by the config file or `TFSCLI_PAT`. For every other
  setting the precedence flag > environment > file > default is unchanged.
- Removing the flag is a breaking change. Before 1.0 that is acceptable; it is
  recorded in `CHANGELOG.md` under `Removed`.
- `skills/tfscli/SKILL.md` keeps a general rule — never pass the token as a
  command argument — rather than a reference to the removed flag.
- How the token is stored (plaintext file, credential helper, OS keychain) is a
  separate discussion and out of scope here.

---

### TASK-01 `remove-pat-flag`
**Description:** Remove the global `--pat` flag from `internal/cli` and the
`PAT` field from `config.Overrides`. The "pat is not set" message stops
suggesting `--pat`. Tests in `internal/cli/cli_test.go` supply the token through
`TFSCLI_PAT` instead of the flag. Update the documentation that mentions the
flag: the settings table in `README.md`; the settings table and "The token"
section in `skills/tfscli/SKILL.md`; the overridability statement in
`docs/project-brief.md`, which gains an exception for the PAT; the config
precedence item in `CLAUDE.md`. Record the change in `CHANGELOG.md` through the
`changelog` skill, section `Removed`.
**Definition of done:** `tfscli --pat x workitem get …` fails with an unknown
flag error. Tests cover the PAT coming from the environment and from the config
file. Outside `docs/archive/` and `CHANGELOG.md`, `--pat` no longer appears in
the repository. `make check` passes.
**Status:** Skipped — superseded by 2026-10-01-tasks-auth-login.md
