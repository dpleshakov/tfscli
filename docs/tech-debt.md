## tech-debt.md

### Active

#### TD-01 `login-anonymous-detection`
**Problem:** `tfscli auth login` treats a `_apis/connectionData` response as an
anonymous answer, and refuses the credential, only when
`authenticatedUser.id` is empty (`internal/cli/auth.go`, `verify`). It is not
verified that a real TFS or Azure DevOps Server answers an anonymous request
that way: the server may instead return the anonymous identity with a
non-empty id, in which case an invalid PAT on a server that admits anonymous
access would be accepted and stored.
**Why deferred:** the server behaviour can only be established against a live
instance, and no live run has happened yet; the stub-server tests cover the
assumed shape only.
**Trigger:** the live check of `auth login` (TASK-04 of
`docs/archive/2026-10-01-tasks-auth-login.md`, done separately), or the
research in `docs/2026-10-01-tasks-login-api-version.md` if it replaces the
verification request.
**Added:** 2026-10-01, in a pre-push review of the unpushed commits

---

### Closed
