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
instance. The live run of release 0.0.5 confirmed that `auth login` with a
valid PAT succeeds, but did not send an anonymous request or an invalid PAT;
the stub-server tests cover the assumed shape only.
**Trigger:** a live run of `auth login` with an invalid PAT, or a request
without credentials to `_apis/connectionData`, against a server that admits
anonymous access.
**Added:** 2026-10-01, in a pre-push review of the unpushed commits

---

### Closed

#### TD-02 `tls-1.2-minimum`
**Fixed:** 2026-10-05 — not a deferred problem but a decision; recorded under
"Defaults are safe" in `docs/architecture.md`

#### TD-03 `request-duration-test`
**Fixed:** 2026-10-05

#### TD-04 `url-userinfo-in-output`
**Fixed:** 2026-10-05

#### TD-05 `collection-named-like-virtual-directory`
**Fixed:** 2026-10-05 — dropped without a change: the naming is not known to
occur, and a report of it would surface as a `not_found` at login

#### TD-06 `path-segments-unescaped`
**Fixed:** 2026-10-05
