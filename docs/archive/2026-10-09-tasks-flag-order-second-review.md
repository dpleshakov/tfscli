# 2026-10-09-tasks-flag-order-second-review.md

**Status:** Archived

## Context

A second code review of the branch `command-shape-for-agent-permissions` found
the following, confirmed on the built binary:

| Invocation | Result |
|---|---|
| `tfscli -p MyProj wit work-items gte 1` | `unknown shorthand flag: 'p' in -p`, where `tfscli wit work-items gte -p MyProj 1` reports `unknown command "gte" ... (did you mean "get"?)` |
| `tfscli --verbose wit wiql query-by-wiql --query "x = 'a' and y = \"b\""` | the corrected command quotes the query as a Go string, `"x = 'a' and y = \"b\""`, which PowerShell does not read back as the same word |
| `tfscli completion bsh` | prints the help of `completion` and exits 0 |

Decisions already made:

- When the walk stops at a group, the flags before the path are not checked
  against it: a group ignores flags it does not know, and they are usually the
  flags of the intended command placed before a mistyped name. The line gets
  the "must come first" error, and the corrected command lets cobra report the
  mistyped name. `--version` before a group is still reported with the way to
  run it.
- The corrected command is shown only when every word can be quoted so that
  bash and PowerShell read it alike: a word of safe characters as it is; a word
  without a single quote (ASCII or typographic) and without a control character
  in single quotes; a word with a single quote in double quotes only when it
  contains no `"`, typographic double quote, `$`, backtick, `\`, `!`, or
  control character. Otherwise the message says to move the command to the
  start and keep every other word in its order, without a command to copy.
- cobra's `completion` group gets the same handling of a leftover argument as
  tfscli's own groups.
- A group other than the root defines no flags at all; the root defines only
  persistent flags, which are used after the full path. The tree test enforces
  exactly that.
- cobra's hidden `__complete` command, with its alias `__completeNoDesc`, is
  named in the brief as the one exception to the flag order rule and to the
  one-form rule: it runs no command, and the words after it are the command
  line to complete.

---

### TASK-01 `group-flags`
**Description:** In `internal/cli/flagorder.go`, check the flags before the
path for unknown or malformed ones only when the command reached has no
subcommands, keeping the `--version` error for a group. Add the mistyped-name
case of the table to `TestFlagBeforeTheCommandPathIsRefused`.
**Definition of done:** `tfscli -p MyProj wit work-items gte 1` gets the "must
come first" error, whose corrected command gets cobra's unknown-command error;
the existing `--version` cases pass; `make check` passes.
**Status:** Done

### TASK-02 `quoting`
**Description:** Rework `shellJoin` in `internal/cli/flagorder.go` per the
decision above, with the message without a command to copy when a word has no
portable form. Add test cases for a word with `'` and `"`, `'` and `$`, `'` and
`\`, a typographic single quote, and a newline.
**Definition of done:** The corrected command is shown exactly when every word
has a portable form; the new tests pass; `make check` passes.
**Status:** Done

### TASK-03 `completion-group`
**Description:** In `internal/cli/cli.go`, share the leftover-argument check of
`newGroupCmd` with cobra's `completion` group, registered in `newRoot`. Cover
`tfscli completion bsh` in `TestUnknownCommandBelowTheRootIsAnError`.
**Definition of done:** `tfscli completion bsh` is an unknown command error
that suggests `bash`; `tfscli completion powershell` still prints the script;
`make check` passes.
**Status:** Done

### TASK-04 `tests`
**Description:** Add `-p=X` forms to `oracleWords` and a hand case with
`-p=MyProject` before the action; extract `isCompletionRequest` and use it in
`findPath` and `checkAgainstCobra`; correct the comment on one tree per worker;
make `TestEveryGroupOnlyGroups` reject any flag defined on a group other than
the root.
**Definition of done:** `shorthands` is fully covered; the tests pass; `make
check` passes.
**Status:** Done

### TASK-05 `docs`
**Description:** Bring the documents in line with TASK-01 to TASK-04: in
`docs/project-brief.md`, name `__complete` and its alias as the exception; in
`docs/architecture.md`, describe the group case, the quoting rule, the
`completion` group, and the stricter tree test, and drop the claim that every
group is built with `newGroupCmd`; in `flagorder.go`, correct the comments on
group flags and on quoting; in `CHANGELOG.md`, shorten the `Changed` entry and
record the silent no-op and `completion` in `Fixed`.
**Definition of done:** The documents agree with the code and with each other
on these points; `make check` passes.
**Status:** Done
