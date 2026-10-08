# 2026-10-08-tasks-command-path-walk.md

**Status:** Active

## Context

The check that refuses flags before the end of the command path
(`docs/archive/2026-10-08-tasks-command-shape-permissions.md`) locates the
target command with cobra's `Find`. `Find` removes flags while it looks for
subcommands, and decides whether a flag takes a value only from the flags of
the command it stands on at that moment (`stripFlags` in `command.go` of cobra
1.10.2). A flag unknown there is assumed to take a value and swallows the next
word. Two kinds of flags are unknown at that point: `--help`, `-h`, and
`--version`, which cobra registers only when it executes the command found,
and the flags of an action placed at the level of its group. Observed on the
built binary:

| Invocation | Word swallowed | Result |
|---|---|---|
| `tfscli --help wit work-items get` | `wit` | `unknown command "work-items"`, exit 1 |
| `tfscli wit --help work-items get` | `work-items` | help of `wit`, not of `get`, exit 0 |
| `tfscli wit wiql --time-precision query-by-wiql --query X` | `query-by-wiql` | help of `wiql`, exit 0, no request |

In the third case `Find` stops at the group, so the check sees the flag after
the path and lets it pass; the group ignores unknown flags
(`FParseErrWhitelist` in `newGroupCmd`) and prints its help with exit 0, a
false success. gh 2.102.0 avoids the first two for `--help` alone by declaring
it as a persistent flag on its root, and still fails on `-h`; it reports a flag
of an action placed before the action as an unknown flag, since its groups do
not ignore unknown flags.

Decisions already made:

- The check walks the arguments from the root itself instead of using `Find`.
  A word naming a subcommand of the current command is a step of the path. A
  word that follows a flag taking a value is that value, even when it names a
  subcommand, as pflag would parse it. Whether a flag takes a value is decided
  by a flag of that name or shorthand on the current command, inherited by it,
  or defined anywhere below it; `--help`, `-h`, and `--version` take none; a
  flag found nowhere is taken to have no value.
- cobra's `help` and `completion` commands are registered when the tree is
  built, so that the walk treats them as ordinary commands and they need no
  exception.
- Every flag before the last word of the path is refused, `--help` and `-h`
  included, with the corrected command. `--version` belongs to the root only,
  so moving it would make an unknown flag; it is refused with a message that
  names `tfscli --version`.
- An invocation without a command path, such as `tfscli --help` or
  `tfscli --version`, is not affected.
- The whitelist of unknown flags in `newGroupCmd` stays: after the check no
  flag reaches a group before a valid action, and the whitelist keeps an
  unknown action name reported as an unknown command.

---

### TASK-01 `path-walk`
**Description:** Replace the use of `Find` in `checkFlagOrder` with the walk
described in the Context section; register the `help` and `completion`
commands in `newRoot`; refuse `--version` before the end of the path with its
own message.
**Definition of done:** Tests cover every row of the table above, `-h` before
the path, a flag of an action placed at the level of its group, `help` and
`completion` with a flag before them, `--version` before a path, and a value
that names a subcommand. `make check` passes.
**Status:** Done

### TASK-02 `docs`
**Description:** Update the wording on exceptions in `docs/project-brief.md`,
`docs/architecture.md`, `CLAUDE.md`, `README.md`, and
`skills/tfscli/SKILL.md`: `--help` follows the command path like any other
flag, and only an invocation without a path, such as `tfscli --help` or
`tfscli --version`, has none. Update the unreleased `CHANGELOG.md` entry to
match.
**Definition of done:** No document states that `--help` may precede the end
of the path, or that `help` and `completion` are exceptions; the architecture
describes the walk instead of `Find`. `make check` passes.
**Status:** Pending
