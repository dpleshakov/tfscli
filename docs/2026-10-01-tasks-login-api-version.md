# 2026-10-01-tasks-login-api-version.md

**Status:** Active

## Context

`tfscli auth login` verifies the credential with a request to
`{url}/_apis/connectionData`. That request is tied to a REST API version,
although the version has nothing to do with what is being verified — only the
URL and PAT pair.

The current behaviour, introduced by `2026-10-01-tasks-auth-login.md`:

- The version is taken from the config file only. `--api-version` is accepted
  on `auth login`, because it is a global flag, but silently ignored, and so is
  `TFSCLI_API_VERSION`.
- Without a config file the version is the built-in default, `7.2`.
- `-preview` is appended to the version, on the unverified assumption that
  `connectionData` is a preview resource and a released version is refused
  for it.

As a result, a login against a server older than API 7.2 is impossible
without a config file, and the error a user sees (an out-of-range version,
HTTP 400, category `config`) does not point at a fix that works for this
command.

---

### TASK-01 `research`
**Description:** Establish how `_apis/connectionData` behaves across TFS and
Azure DevOps Server versions: with no `api-version` at all, with a released
version, with a `-preview` version, and with a version newer than the server
supports. Also establish:
- whether the response carries `authenticatedUser.id` in the same shape on
  older servers, since `verify` in `internal/cli/auth.go` relies on it (see
  TD-01 in `docs/tech-debt.md` for the anonymous case);
- whether `connectionData` also answers when the URL entered at login already
  contains a collection, such as `https://tfs.company.com:8080/tfs/DefaultCollection`.
  If it does, login accepts and stores that URL, every later command appends
  the collection a second time, and the user receives a bare `not_found` with
  no hint at the cause.

Compare candidate replacement requests, such as
`{collection}/_apis/projects?$top=1`, by availability across server versions,
by whether they need a collection, by whether they distinguish an invalid
PAT from anonymous access, and by whether they distinguish a server URL from
a collection URL. Sources: Microsoft documentation, the official client SDKs,
and a live server where one is available.
**Definition of done:** The findings are recorded in the Context section,
each with its source; anything not confirmed is marked as unverified.
**Status:** Pending

### TASK-02 `default-version-research`
**Description:** Establish whether tfscli needs to send `api-version` on
every request, and what the default should be, in the light of the "Server
compatibility is preserved" principle in `docs/project-brief.md`. The points
and contradictions known at planning time:
- Every request carries `api-version` (`internal/apiclient/apiclient.go`), and
  the built-in default is `7.2` (`internal/config/config.go`). If only the
  newest Azure DevOps Server release supports 7.2, every command fails on
  older servers when nothing is configured. Unverified: the mapping of server
  releases to REST API versions is not recorded anywhere in the project.
- `docs/project-brief.md` names the configurable API version as the means to
  support older TFS installations, yet nothing tells the user which version
  the server supports or that the version has to be changed. An out-of-range
  version is reported as a `config` error carrying the server's message, which
  probably names the latest version the server supports but does not name
  `--api-version`, `TFSCLI_API_VERSION`, or `apiVersion` as the fix.
- An empty value is replaced by the default (`internal/config/config.go`), so
  a request without `api-version` cannot be made even explicitly.
- Microsoft documentation describes `api-version` as required on every
  request. Unverified: the server rejects a request without it for some
  methods ("No api-version was supplied ..."), while a GET request may be
  served at the latest version. If a GET without a version works on all
  target servers, a read-only client may not need to send one.
- An explicit version pins the shape of the response; without one the shape
  is determined by the server and may differ between servers. This matters
  little for the markdown output of selected fields, but it matters for the
  planned `--json` output, which is meant to be a stable contract.
- The paths and the response shape used by `workitem get`, including the
  project segment in `_apis/wit/workitems`, have not been checked against
  any server older than the default version.
- Neither `README.md` nor `skills/tfscli/SKILL.md` tells the user how to find
  the API version their server supports or how it relates to the server
  release; both list only the default `7.2`.
- Alternatives to compare: no `api-version` at all; no version by default,
  with `--api-version`, `TFSCLI_API_VERSION`, and `apiVersion` as overrides;
  the lowest version all target servers support as the default; on an
  out-of-range error, reading the latest supported version from the server's
  message and either retrying with it or naming it in the error.
Sources: Microsoft documentation, the official client SDKs, and a live server
where one is available.
**Definition of done:** The findings are recorded in the Context section, each
with its source; anything not confirmed is marked as unverified. The
alternatives are compared by server compatibility, stability of the response
shape, and whether tfscli works with no configuration.
**Status:** Pending

### TASK-03 `decide`
**Description:** Using the findings, decide how `auth login` handles the API
version. Options known at planning time: send no `api-version` for the
verification request; honour `--api-version` and `TFSCLI_API_VERSION` as other
commands do; try without a version first and with one second; replace the
verification request. The decision has to satisfy the "Server compatibility
is preserved" and "Works out of the box; errors lead the way" principles in
`docs/project-brief.md`. In particular, an API version the server does not
support must produce an error that names the next step, and that step must
work for the command that failed: today the error for an out-of-range version
carries only the server's message, and for `auth login` the only working fix,
the config file, is named nowhere. Record the decision and the rejected
options with their reasons in the Context section.
**Definition of done:** The decision and the rejected options are recorded in
the Context section, and implementation tasks for the decision are added to
this file.
**Status:** Pending
