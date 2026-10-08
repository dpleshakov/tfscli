# 2026-10-08-tasks-command-shape-permissions.md

**Status:** Active

## Context

Goal: the command line of tfscli is easy to govern with the permission rules
of AI agent hosts, such as Claude Code and opencode, now and after write
operations are added. Those rules match the text of a command: Claude Code
evaluates deny, then ask, then allow, and rule specificity does not change the
order; opencode applies the last matching rule. A rule written as a prefix
(`tfscli wit work-items get *`) is reliable only if the command path always
opens the command line, and no rule can tell a writing command from a reading
one by its action name, since action names come from the REST API and form an
open set (`add`, `create`, `update`, `delete`, `queue`, `run`, ...), while some
POST operations, such as `get-batch` and `query-by-wiql`, only read.

Decisions already made:

- **Flags follow the command path.** A token starting with `-` before the last
  word of the command path is refused with a `config` error that shows the
  corrected command. Today cobra accepts `tfscli --verbose wit work-items get 1`,
  which a prefix rule does not match. Exceptions: `--help` and `--version` at
  any level, and cobra's own `help` and `completion` commands. This is a
  breaking change to the command line.
- **One form per command.** No command aliases, no abbreviated command or flag
  names, no cobra prefix matching.
- **Local commands do not take API area names.** The names of local commands
  (`auth`, `licenses`, `help`, `completion`) must not coincide with an area of
  the REST API reference.
- **Whether a command changes server state is a property of the command**, not
  of its flags: a reading command never gains a side effect through a flag. No
  generic pass-through command (`invoke`, `api`) with an arbitrary method is
  added; if one ever is, it falls under the gate flags below.
- **Gate flags for future write operations.** Every operation that changes
  server state requires `--allow-changes`. An irreversible operation requires
  both `--allow-changes` and `--allow-irreversible`, so that one ask rule on
  `--allow-changes` covers every change and a deny rule on
  `--allow-irreversible` excludes irreversible ones. The classification follows
  the meaning of the operation, not its HTTP method.
- **Criterion of irreversibility:** an operation is irreversible when its
  effect cannot be undone by another operation of the REST API. Deleting a work
  item to the recycle bin is reversible (it can be restored); destroying it is
  not.
- **The gate flags are flags only.** No environment variable or config key
  supplies them, as an exception to the precedence chain, like the credential:
  a setting outside the command line is invisible to permission rules. Short
  forms are not provided.
- **The gate is checked before any request**, the `OPTIONS` negotiation
  included. A missing gate flag is a `config` error naming the flag; no new
  error category is added.
- **The gate covers server state only.** Local commands that change local
  state live under `auth`, are interactive, and are excluded by a rule on
  `tfscli auth *`.
- The gate flags deviate from "Follow TFS API structure"; the justification is
  safety under agent permission rules, and it is recorded in the brief.
- Claude Code's documented matching was checked
  (https://code.claude.com/docs/en/permissions); opencode's handling of
  compound commands is not documented on https://opencode.ai/docs/permissions/
  and is not claimed in user documentation.

---

### TASK-01 `brief`
**Description:** In `docs/project-brief.md`: add to Design Principles the rules
of the command line shape — flags after the command path with the exceptions
above, one form per command, local command names distinct from API areas,
server-state changes as a property of the command, no generic pass-through
command. Extend Non-Goals → Write operations with the gate flags: both flags,
the criterion of irreversibility, flags only, the check before any request,
the `config` error, and the justification of the deviation from the API
structure. Record in Configuration that the gate flags are outside the
precedence chain.
**Definition of done:** Every decision in the Context section that concerns the
product is stated in the brief; the brief does not contradict itself on
configuration precedence.
**Status:** Done

### TASK-02 `architecture`
**Description:** In `docs/architecture.md`: describe where the check of flags
before the command path lives — an explicit check in `run()` built on
`root.Find`, since cobra accepts flags at any position and a
`PersistentPreRunE` on the root is silently replaced by a hook of the same
name on a child command; the tree-walking guard test; and, under "Notes for
future evolution", how writing commands will be marked so that the guard test
can require the gate flags of them.
**Definition of done:** The cli module description and the notes for future
evolution cover the three points.
**Status:** Pending

### TASK-03 `claude-md`
**Description:** In `CLAUDE.md`: add short items to "Non-negotiable product
constraints" (gate flags, flags only, one form per command) and to "Command
shape" (flags after the command path), referring to the brief for the full
rules.
**Definition of done:** A reader of `CLAUDE.md` alone learns that flags follow
the command path, that aliases are excluded, and that write operations require
the gate flags.
**Status:** Pending

### TASK-04 `flags-after-command`
**Description:** In `internal/cli`, refuse a token starting with `-` before the
last word of the command path with a `config` error that names the flag and
shows the corrected command, with the exceptions recorded in the Context
section.
**Definition of done:** Tests cover the refusal of `--verbose` and
`--api-version` placed before the path, at the root and between path words, and
the unchanged behaviour of `--version`, `--help` at every level, `help ...`,
and `completion ...`. `make check` passes.
**Status:** Pending

### TASK-05 `command-tree-guard`
**Description:** Add a test that walks the whole command tree and fails when
any command declares `Aliases` or when `cobra.EnablePrefixMatching` is on.
**Definition of done:** The test fails when an alias is added to any command
and passes on the current tree. `make check` passes.
**Status:** Pending

### TASK-06 `user-docs`
**Description:** In `README.md`, state under Usage that flags follow the
command, and add to "Use with an AI agent" permission rules for the current
read-only commands for Claude Code (Bash and PowerShell) and opencode: allow
`tfscli wit *` and `tfscli licenses`, deny `tfscli auth *`. In
`skills/tfscli/SKILL.md`, state that flags follow the command. Add an entry to
`CHANGELOG.md` per the `changelog` skill for the refusal of flags before the
command path, marked as a breaking change.
**Definition of done:** The README, the agent skill, and the changelog describe
the new rule; the README claims nothing about opencode beyond its documented
matching. `make check` passes.
**Status:** Pending
