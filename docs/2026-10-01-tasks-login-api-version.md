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

### Findings: `connectionData` and alternatives

No live server was available; every finding comes from Microsoft
documentation or the official client SDKs. Server behaviour inferred from
SDK code is what the SDK authors rely on, not an observation, and is marked
as such.

Sources referred to below:

- [S1] REST API versioning, Microsoft Learn:
  https://learn.microsoft.com/en-us/azure/devops/integrate/concepts/rest-api-versioning
- [S2] azure-devops-python-api, `location_client.py` and
  `models/connection_data.py` at tags `0.1.22` (API 4.0, 4.1) and `5.0.0`
  (5.0, 5.1), and branches `dev` (7.1) and `dev7.2` (7.2):
  https://github.com/microsoft/azure-devops-python-api
- [S3] azure-devops-go-api, `azuredevops/v7/location/client.go`,
  `location/models.go`, `client.go`, `versionnegotiation.go` on branch `dev`:
  https://github.com/microsoft/azure-devops-go-api
- [S4] azure-devops-node-api, `api/WebApi.ts` at tag `v6.6.2` and on
  `master`, and the "API and TFS Mapping" table in `README.md`:
  https://github.com/microsoft/azure-devops-node-api
- [S5] TFS 2017 REST API reference (archived), Projects and Project
  Collections:
  https://learn.microsoft.com/en-us/previous-versions/azure/devops/integrate/previous-apis/tfs/projects?view=tfs-2017,
  https://learn.microsoft.com/en-us/previous-versions/azure/devops/integrate/previous-apis/tfs/project-collections?view=tfs-2017
- [S6] User reports of the out-of-range error, e.g.
  https://github.com/maikvandergaag/msft-extensions/issues/329,
  https://github.com/cribeiro84/azure-devops-pull-request-hub/issues/118
- [S7] The "No api-version was supplied" error for a POST request:
  https://dexterposh.github.io/posts/002-azdo-tip-api/

**Server releases and API versions.** [S1] maps them as: TFS 2015 — up to
2.x; TFS 2017 — 3.x; TFS 2018 — 4.x; Azure DevOps Server 2019 — 5.x; 2020 —
6.x; 2022 — 7.x. The table stops at 7.0; which releases support 7.1 and 7.2
is not stated there (unverified). [S1] also states that "API version **must**
be specified with every request".

**`connectionData` is a preview resource in every SDK version.** The Python
SDK calls it with `4.0-preview.1`, `4.1-preview.1`, `5.0-preview.1`,
`5.1-preview.1`, `7.1-preview.1`, and `7.2-preview.1`, each marked
"[Preview API]" [S2]; the Go SDK uses `7.1-preview.1` [S3]. The resource has
no released version at any API level the SDKs cover. It is absent from the
current REST API reference.

**A released version for it.** The Go SDK appends `-preview` when a resource
has no released version, or a released version lower than the one requested
(`negotiateRequestVersion`) [S3]. This implies that the server refuses a
released version for a preview resource, which is the assumption in
`internal/cli/auth.go`. Not observed on a server: unverified.

**A preview version over time.** [S1]: after an API is released, its preview
version is deprecated and "can be deactivated after 12 weeks", after which
requests specifying `-preview` are rejected. `connectionData` has not been
released at any level so far, but a client that always appends `-preview` to
a configured version depends on that remaining true.

**No `api-version` at all.** Contradicting [S1], two SDKs rely on the server
accepting it:
- The Node SDK's `WebApi.connect()` sends
  `GET {url}/_apis/connectionData` with no `api-version`, in the query or in
  the `Accept` header, both at `v6.6.2`, which its README maps to TFS 2018
  Update 2, and on `master` [S4].
- The Go SDK's negotiation passes an empty version through, with the comment
  "if no api-version is sent to the server, the server will decide the
  version. The server uses the latest released version if the endpoint has
  been released, otherwise it will use the latest preview version" [S3].

The error "No api-version was supplied for the \"POST\" request" is
documented for POST [S7] and reported for PUT; no report of it for GET was
found. Conclusion: a GET of `connectionData` without `api-version` is what
Microsoft's Node SDK does against TFS 2018 Update 2 and later. Behaviour on
TFS 2017 and earlier: unverified.

**A version newer than the server supports.** The server answers
"The requested REST API version of 6.1 is out of range for this server. The
latest REST API version this server supports is 6.0." [S6], which tfscli
reports as `config` with that text. The SDKs avoid it by negotiating:
`OPTIONS {url}/_apis` returns, per resource, `minVersion`, `maxVersion`,
`releasedVersion`, and `resourceVersion`, and a requested version above
`maxVersion` is lowered to it, with `-preview` added when `maxVersion` is not
released [S3]. Whether that `OPTIONS` request needs authentication, and
whether TFS 2017 answers it: unverified.

**Response shape on older servers.** `ConnectionData` has the same fields at
API 4.0 as now — `authenticatedUser`, `authorizedUser`, `deploymentId`,
`instanceId`, `lastUserAccess`, `locationServiceData`,
`webApplicationRelativeDirectory` — and the 4.0 `Identity` carries `id` and
`providerDisplayName` [S2]. `deploymentType` was added later [S3]. So
`verify` in `internal/cli/auth.go` reads fields present since TFS 2018. Before
4.0 (TFS 2017, 2015): unverified. How an anonymous request is answered
remains open (TD-01).

**A URL that already contains a collection.** The SDKs call `connectionData`
on the base URL they are given, which on-premises is the collection URL
[S3, S4], so the resource answers at collection level as well as at server
level. A login with a collection URL is therefore accepted, as feared. The
model describes `deploymentId` as "the id for the server" and `instanceId` as
"the instance id for this host" [S3]; at server level the two would
presumably be equal and at collection level differ, which would tell the two
URLs apart. Unverified. `webApplicationRelativeDirectory` names the virtual
directory of the host and may serve the same purpose; unverified.

**Candidate replacement requests:**

| Request | Available since | Needs a collection | Identifies the user | Tells server URL from collection URL |
|---|---|---|---|---|
| `{url}/_apis/connectionData` | TFS 2018 per SDKs; earlier unverified | No | Yes, `authenticatedUser` | Possibly, via `instanceId` and `deploymentId` (unverified) |
| `{url}/{collection}/_apis/projects?$top=1` | API 1.0, released, TFS 2015 [S5] | Yes | No | No: it succeeds whenever the full URL is a collection |
| `{url}/_apis/projectCollections` | API 1.0, as `1.0-preview.2`, TFS 2015 [S5] | No | No | Unverified how it answers on a collection URL |
| `OPTIONS {url}/_apis` | Used by the Go and Python SDKs [S2, S3] | No | No | No |

`projects` needs a collection, which `auth login` does not have: the
collection is not part of the credential. None of the alternatives
identifies the user, so none can tell an invalid PAT from anonymous access by
its body, and none supports the "Logged in to … as …" message. Whether an
invalid PAT on any of them yields 401 rather than anonymous access is the
open question of TD-01.

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
**Status:** Done

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
