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

#### TD-02 `tls-1.2-minimum`
**Problem:** tfscli connects only with TLS 1.2 or later: `MinVersion` is set
to TLS 1.2 when `caBundle` or `insecureSkipVerify` is configured
(`internal/apiclient/apiclient.go`, `newTLSConfig`), and the Go client default
is the same otherwise. A server that offers only TLS 1.0 or 1.1, such as an
older TFS on an older Windows Server without TLS 1.2 enabled, cannot be
reached. This restricts server compatibility, and the "Server compatibility
is preserved" principle in `docs/project-brief.md` requires such a restriction
to be recorded with its justification.
**Why deferred:** TLS 1.0 and 1.1 are deprecated and insecure, and the token
travels in every request; accepting them would weaken every connection for
the sake of servers that are not known to exist among the users. No such
server has been reported.
**Trigger:** a user reports a server that cannot negotiate TLS 1.2.
**Added:** 2026-10-04, in a conversation reviewing the repository against the
server compatibility and out-of-the-box principles

#### TD-03 `request-duration-test`
**Problem:** `TestLogsEveryRequestExactlyOnce` in
`internal/apiclient/apiclient_test.go` asserted that the logged request
duration is positive. On Windows a request to a local `httptest` server often
measures exactly `0s`, so the test failed in most local runs of `make check`,
on `main` as well. The assertion is replaced by `t.Skip` until this is
resolved, so a zero or negative duration is currently not detected.
**Why deferred:** the cause is presumed, not established — the resolution of
the Windows clock as seen through `time.Since`, or a local round trip shorter
than it — and the fix depends on it: a server handler that waits, an
injected clock in `loggingTransport`, or a weaker assertion such as
`dur >= 0`. None of this concerns the work in which the failure was noticed.
**Trigger:** the next change to `loggingTransport` or to request logging, or
any further flaky failure in the `apiclient` tests.
**Added:** 2026-10-04, while executing TASK-04 of
`docs/archive/2026-10-01-tasks-login-api-version.md`

#### TD-04 `url-userinfo-in-output`
**Problem:** `config.NormalizeURL` (`internal/config/auth.go`) keeps a
`user:password@` part of the server URL, and three messages print the URL as
stored: the `Logged in to …` line and the notice that the collection was
removed from the URL in `promptAuth`, and the 404 message built by
`notFoundAt` (`internal/cli/auth.go`). A password typed into the login URL
would therefore reach stdout or stderr. The request logging and the network
errors in `internal/apiclient/apiclient.go` already redact it.
**Why deferred:** the URL is typed by the user into an interactive prompt and
is printed back only to that user's terminal; credentials in the URL serve no
purpose with PAT authentication and are not expected in practice.
**Trigger:** any further message or log line that prints the stored URL, or a
report of userinfo in a server URL. The likely fix is to reject or strip
userinfo in `NormalizeURL`, which covers every place at once.
**Added:** 2026-10-04, in review of
`docs/archive/2026-10-04-tasks-login-collection.md`

#### TD-05 `collection-named-like-virtual-directory`
**Problem:** `trimCollection` (`internal/cli/auth.go`) removes the last path
segment of the server URL entered at `tfscli auth login` whenever it equals
the collection, case ignored. On a server whose virtual directory has the
same name as the collection, such as a collection `tfs` on
`https://host/tfs`, the correct URL loses its segment, the login check
reports `not_found`, and neither that message nor the notice explains how to
keep the segment (entering it twice, or using `TFSCLI_AUTH`).
**Why deferred:** such a naming is not known to occur on a real server, and
the workaround exists, although it is undocumented.
**Trigger:** a report of a server where the virtual directory and the
collection share a name, or the next change to the login prompts. Options:
mention in the notice how to keep the segment, or retry once with the
untrimmed URL after a 404.
**Added:** 2026-10-04, in review of
`docs/archive/2026-10-04-tasks-login-collection.md`

#### TD-06 `path-segments-unescaped`
**Problem:** the domain packages join the project and the team into the
request path as given (`internal/workitem/workitem.go`,
`internal/wiql/wiql.go`, `path`), and `apiclient` adds the path with
`url.URL.JoinPath`, which treats it as already escaped. Spaces, `#`, and `?`
are encoded correctly, but `%` is not: a name with an invalid escape
sequence, such as `100% Done`, makes `JoinPath` drop the whole path silently,
so the request goes to the collection URL itself; a name with a valid
sequence, such as `A%20B`, is decoded and addresses `A B` instead. It is not
known whether TFS permits `%` in project or team names.
**Why deferred:** no such name is known to occur, the behaviour predates
`--team`, and the fix touches every domain package and the contract of
`APIClient.Get` and `APIClient.Post`, which concerns more than the work in
which it was noticed. The likely fix is to escape each user-supplied segment
with `url.PathEscape`, either in the domain packages or by having `apiclient`
take the path as segments.
**Trigger:** the next domain that puts a user-supplied name into the path,
such as a repository name for Git pull requests, or a report of a project or
team name containing `%`.
**Added:** 2026-10-04, in review of
`docs/archive/2026-10-04-tasks-domains-2-wiql.md`

---

### Closed
