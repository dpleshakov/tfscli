# 2026-10-09-tasks-simple-path-rule.md

**Status:** Archived

## Context

The flag order check locates the command twice: by a walk that repeats
pflag's rules for flag values, and by cobra's `Find`, taking the deeper of the
two (`docs/archive/2026-10-09-tasks-flag-order-review.md`). Both defects found
so far were divergences between the walk and pflag; the cross-check covers
them, at the cost of two parsers guarding each other.

The shape of the command tree makes knowledge of flag values unnecessary. A
command with subcommands only groups them: it takes no arguments, defines no
flags of its own, and does nothing but print its help; only a command without
subcommands runs. In a valid command line, therefore, every word that names a
subcommand of the command reached so far is a step of the path, and no such
word can follow the last one: the last step has no subcommands.

Decisions already made:

- The command is the one reached from the root by taking, in order, every word
  that names a subcommand of the command reached so far and skipping every
  other word, up to `--`. Its path must occupy the first words of the command
  line; otherwise the line is refused with the corrected command, as now. This
  reaches at least as deep as `Find`, which descends only through such words,
  so neither the walk that repeats pflag's rules nor the cross-check with
  `Find` is needed.
- The only ambiguity left is a value placed before the path that names a
  subcommand, as in `wit work-items -p get get 1`; such a line is invalid in
  any case and is refused.
- Unknown flags before the path are still reported as unknown, looked up on
  the command found; `--version` before a path keeps its own message; the
  quoting of the corrected command is unchanged.
- The shape of the tree becomes a rule of the brief and is enforced by the
  command tree test: a command with subcommands accepts no argument and
  defines no flag of its own. Persistent flags stay allowed, since they are
  used after the full path. A group that runs on its own, as `git stash` or
  `git remote` do, is excluded.

---

### TASK-01 `simple-path`
**Description:** In `internal/cli/flagorder.go`, replace `walkPath`,
`takesValue`, `depth`, and the cross-check with `Find` by the rule above. In
`internal/cli/tree_test.go`, require of every command with subcommands that it
rejects an argument and defines no local non-persistent flag. The test found
that `auth` was built as a plain cobra command rather than with
`newGroupCmd`, so `tfscli auth bogus` printed its help with exit 0; build it
with `newGroupCmd`.
**Definition of done:** Every case in `flagorder_test.go` passes unchanged;
the tree test fails when a group is given a local flag or accepts an
argument, and passes on the current tree. `make check` passes.
**Status:** Done

### TASK-02 `docs`
**Description:** Add the rule on groups to "The command line can be governed
by permission rules" in `docs/project-brief.md`, to the matching items in
`CLAUDE.md` and `CONTRIBUTING.md`; describe the simpler check and its
dependence on the shape of the tree in `docs/architecture.md`. Record in
`CHANGELOG.md` that an unknown command under `tfscli auth` is now an error.
**Definition of done:** The documents state that a command with subcommands
only groups them, and the architecture no longer describes the walk or the
cross-check with `Find`. `make check` passes.
**Status:** Done
