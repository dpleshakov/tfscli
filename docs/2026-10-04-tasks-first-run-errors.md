# 2026-10-04-tasks-first-run-errors.md

**Status:** Active

## Context

A review of the repository against the "Works out of the box; errors lead the
way" principle in `docs/project-brief.md` found errors that a first-time user
can encounter without being told the next step, and one setting that may be
required without need. The problems tied to the REST API version and to the
login verification request are tracked in
`docs/archive/2026-10-01-tasks-login-api-version.md` and are not repeated here.

The error categories are a contract and stay as they are; the tasks below
change message text only, unless a task records otherwise.

---

### TASK-01 `tls-errors`
**Description:** A failed certificate verification, such as a server behind
an internal CA, falls into the generic transport branch of `transportError`
in `internal/apiclient/apiclient.go` and is reported as
`cannot reach https://…`; the cause is not printed (`tfserr.Print`). The user
is told the server is unreachable, while the fix is `caBundle`, or
`insecureSkipVerify`, in a config file the user does not know about. Report
certificate verification failures, which Go exposes as distinct x509 error
types, with a message that says the certificate was not trusted and names
`caBundle` and the path of the config file. The category stays `network`.
**Definition of done:** An untrusted certificate produces a message naming
the certificate problem, `caBundle`, and the config file path; other
transport failures keep their current messages. Tests cover both. `make check`
passes.
**Status:** Pending

### TASK-02 `http-error-steps`
**Description:** The generic messages for HTTP 401, 403, and 404 in
`categoryFor` (`internal/apiclient/apiclient.go`) state the fault but not the
next step. Add one: for 401, issue a new PAT and run `tfscli auth login`, or
update `TFSCLI_AUTH` when the credential came from it; for 403, check that the
PAT has the Work Items (Read) scope and that the user has access to the
project; for 404, check the work item id, the project, and the collection.
Decide whether the step is also appended when the server supplies its own
message, which today replaces the generic wording.
**Definition of done:** Each of the three statuses produces a message with a
next step, the 401 step depends on the credential source, and the decision on
server-supplied messages is recorded in the Context section. Tests cover each
case. `make check` passes.
**Status:** Pending

### TASK-03 `login-messages`
**Description:** Two messages around `auth login` lack a usable next step.
`not logged in: … (run "tfscli auth login", or set TFSCLI_AUTH)`
(`internal/config/auth.go`) does not say that `auth login` is for a person to
run in a terminal, so an agent may try to run it itself.
`tfscli auth login needs an interactive terminal (set TFSCLI_AUTH instead)`
(`internal/cli/auth.go`) does not give the format of the variable, and under
Git Bash (mintty), where stdin is not a console, the right step is another
terminal or `winpty`, which only `README.md` mentions.
**Definition of done:** The first message says that a person runs
`tfscli auth login`; the second gives the `TFSCLI_AUTH` format and mentions
running from another terminal or with `winpty`. Tests are updated. The
representative messages in `skills/tfscli/SKILL.md` match. `make check`
passes.
**Status:** Pending

### TASK-04 `unexpected-response`
**Description:** When the stored URL points at something other than TFS, such
as a proxy or SSO page answering with HTML, the user receives
`TFS returned a response that could not be parsed`
(`internal/workitem/workitem.go`) or
`TFS returned an unexpected response to the login check`
(`internal/cli/auth.go`), with no hint to check the URL. Add that hint, naming
the URL in use.
**Definition of done:** Both messages name the URL and suggest checking it;
for an ordinary command the suggestion includes re-running
`tfscli auth login`. Tests are updated. `make check` passes.
**Status:** Pending

### TASK-05 `minor-messages`
**Description:** `project is not set (… add "project" to the config file)`
(`internal/config/config.go`, `RequireProject`) does not give the path of the
config file, unlike the message for a missing collection. Errors raised by
cobra itself, such as `accepts 1 arg(s), received 0` or an unknown flag, are
printed without the usage text (`SilenceUsage` in `internal/cli/cli.go`) and
without a pointer to `--help`.
**Definition of done:** The project message names the config file path, and
cobra errors end with a pointer to the command's `--help`. Tests cover both.
`make check` passes.
**Status:** Pending

### TASK-06 `project-optional-research`
**Description:** `wit work-items get` requires a project (`-p`, `TFSCLI_PROJECT`,
or the config file), and the request path includes it
(`{project}/_apis/wit/workitems/{id}` in `internal/workitem/workitem.go`).
Unverified: the Get Work Item documentation lists the project as optional,
since a work item id is unique within a collection. If so, tfscli requires a
setting the API does not need, which conflicts with both the out-of-the-box
principle and mirroring the API. Establish whether the project is optional,
on which REST API versions the path without a project and the path with one
are available, and whether the responses differ.
**Definition of done:** The findings are recorded in the Context section, each
with its source; anything not confirmed is marked as unverified.
**Status:** Pending

### TASK-07 `project-optional-decide`
**Description:** Using the findings on the project in the request path, decide whether `-p` becomes
optional for `wit work-items get`, taking server compatibility into account.
**Definition of done:** The decision and the rejected options are recorded in
the Context section, and implementation tasks for the decision are added to
this file.
**Status:** Pending
