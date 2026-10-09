# 2026-10-09-tasks-flag-order-review.md

**Status:** Active

## Context

A code review of the branch `command-shape-for-agent-permissions` found that
the flag order check (`internal/cli/flagorder.go`) can still be bypassed, and
that the documents promise more than the permission rules give. Confirmed on
the built binary against an unreachable server:

| Invocation | Result |
|---|---|
| `tfscli wit work-items --fields -p get 5 -p P` | passes the check; `Error [network]`, so the request was sent |
| `tfscli --help X wit work-items get` | passes the check; cobra prints the help of `get` |
| `tfscli - wit work-items get` | passes the check; `get` receives `-` as its id |
| `tfscli "" wit work-items get 5` | passes the check; `get` receives two arguments |

The walk read a flag value starting with `-` as a new flag, and stopped at a
word cobra skips (`""`, `-`) or at a word cobra takes as the value of a flag it
does not know (`--help X`). It is a second parser of the command line, and it
diverges from cobra's in edge cases.

Decisions already made:

- The walk reads the word after a flag that takes a value as that value before
  anything else, whatever it starts with, as pflag does.
- The command is located both by the walk and by cobra's `Find`, and the
  deeper of the two is taken: the walk catches `Find` stopping at a group, and
  `Find` catches the walk stopping early. The check passes only when the path
  of that command occupies exactly the first words of the command line.
- The corrected command is the path followed by the remaining words in their
  original order. The message names the command and shows the corrected
  command; it no longer lists the misplaced flags, since a stray word such as
  `""` is not a flag and an attached short value made the list inaccurate.
- A flag before the path that the target command does not define is reported
  as unknown, in cobra's wording, instead of being moved into a corrected
  command that would fail the same way. The words before the path are read by
  pflag's rules, so a value such as `-p` after `--fields` is not taken for a
  flag.
- The corrected command quotes every word outside a safe set of characters in
  single quotes, which bash and PowerShell read alike, and falls back to
  double quotes only for a word containing a single quote.
- The README states that a command matching no rule gets the host's default
  (Claude Code asks; opencode allows unless a catch-all `"*": "ask"` is set),
  so the deny rule guards the usual form of the call and is not a security
  boundary. Its examples add an ask rule on `--allow-changes`, so that future
  write commands under `tfscli wit *` are not allowed by the same rule.

---

### TASK-01 `cross-check`
**Description:** Implement the decisions above in `internal/cli/flagorder.go`:
read a pending value before a flag, cross-check the walk with `Find`, build the
corrected command from the remaining words, report unknown flags before the
path as unknown, quote the corrected command for the shell, and make the long
and short lookups in `takesValue` exclusive branches. In
`internal/cli/flagorder_test.go`, add the four invocations of the table, an
unknown flag before the path, a value that starts with `-`, and quoting of a
word with `&` and `[`; check that stderr is empty on the accepted invocations.
**Definition of done:** Every invocation of the table is refused before any
request; the tests listed pass; `make check` passes.
**Status:** Done

### TASK-02 `docs`
**Description:** In `README.md`: describe the fallback to the host's default
for an unmatched command and its consequence for the deny rule; add the ask
rule on `--allow-changes` to the Claude Code and opencode examples; state that
every command starts with its full name without the area/resource/action
template; note the separate message for `--version` before a command. In
`CONTRIBUTING.md`, add the command line rules and the gate flags to the list
in "Before changing behaviour", and the gate flags to the precedence item; do
the same for the precedence item in `CLAUDE.md`. In `docs/architecture.md`,
describe the cross-check with `Find`, unknown flags, and quoting.
**Definition of done:** The documents agree with the code and with each other
on the points above. `make check` passes.
**Status:** Pending
