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
supports. Compare candidate replacement requests, such as
`{collection}/_apis/projects?$top=1`, by availability across server versions,
by whether they need a collection, and by whether they distinguish an invalid
PAT from anonymous access. Sources: Microsoft documentation, the official
client SDKs, and a live server where one is available.
**Definition of done:** The findings are recorded in the Context section,
each with its source; anything not confirmed is marked as unverified.
**Status:** Pending

### TASK-02 `decide`
**Description:** Using the findings, decide how `auth login` handles the API
version. Options known at planning time: send no `api-version` for the
verification request; honour `--api-version` and `TFSCLI_API_VERSION` as other
commands do; try without a version first and with one second; replace the
verification request. Record the decision and the rejected options with their
reasons in the Context section.
**Definition of done:** The decision and the rejected options are recorded in
the Context section, and implementation tasks for the decision are added to
this file.
**Status:** Pending
