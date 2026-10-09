# 2026-10-09-tasks-flag-order-oracle.md

**Status:** Active

## Context

The tests of the flag order check compare it with expectations written by
hand. The defects found in review were cases where cobra treated a command
line differently from what those expectations assumed, so the tests could not
catch them. This work adds a test that takes cobra itself as the oracle and
runs it over command lines generated rather than chosen.

The behaviour is already decided; the test restates the rule in a form that
can be checked on any command line:

- If `checkFlagOrder` accepts a command line, the command cobra selects for it
  (`ExecuteC`, with every command's action replaced by a stub) has its path in
  the first words of the line.
- If `checkFlagOrder` refuses a command line because the path does not come
  first, the corrected command line it proposes is accepted by the check, and
  cobra selects for it the command named in the message.

The converse of the first property does not hold by design: a line such as
`tfscli --verbose wit work-items get 1` is refused although cobra would run
it.

Command lines are built from an alphabet derived from the command tree (every
command name and every flag in its long, short, and inline forms) and a fixed
set of special words (`--`, `-`, the empty word, `--help`, `-h`,
`--version`, an unknown flag, a plain word, a number). All lines up to
length 3 are checked exhaustively, and a fixed-seed random sample covers
lengths 4 to 7, so the test stays deterministic.

While preparing this, shell completion was found broken on this branch:
cobra adds its hidden `__complete` command only when it executes the tree, so
`tfscli __complete wit ""`, which the completion scripts run, is refused as a
misplaced path. The defect was never released, so it needs no changelog entry.

---

### TASK-01 `completion-request`
**Description:** In `checkFlagOrder`, accept a command line whose first word
is `cobra.ShellCompRequestCmd` or `cobra.ShellCompNoDescRequestCmd`: such a
request only lists completions and runs no command. Add a test case and a
sentence on it to the cli module paragraph of `docs/architecture.md`.
**Definition of done:** `tfscli __complete wit ""` lists the subcommands of
`wit` with exit 0, covered by a test. `make check` passes.
**Status:** Done

### TASK-02 `oracle-test`
**Description:** Add the test described in Context to
`internal/cli/flagorder_test.go`, building a fresh command tree for every
command line. Factor the construction of the corrected command line out of
`checkFlagOrder` so that the test can run it as words rather than parse the
message. Mention the test in the cli module paragraph of
`docs/architecture.md`.
**Definition of done:** The test passes on the current code and fails when
the check is weakened, for example when it stops at the first word that is
not a subcommand. `make check` passes, and the test adds no more than a few
seconds to it.
**Status:** Pending
