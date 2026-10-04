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

### Findings: the default API version

Further sources:

- [S8] REST API reference overview, including "API and TFS version mapping":
  https://learn.microsoft.com/en-us/rest/api/azure/devops/?view=azure-devops-rest-7.2
- [S9] Work Items - Get Work Item, views `azure-devops-rest-7.2`,
  `azure-devops-server-rest-7.1`, `azure-devops-server-rest-5.0`,
  `vsts-rest-tfs-4.1`:
  https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/work-items/get-work-item
- [S10] TFS 2017 REST API reference (archived), Work Items, API 1.0:
  https://learn.microsoft.com/en-us/previous-versions/azure/devops/integrate/previous-apis/wit/work-items?view=tfs-2017

**Server releases and API versions** [S8], which supersedes the shorter
table in [S1]:

| Server | REST API version |
|---|---|
| Azure DevOps Server vNext | 7.2 |
| Azure DevOps Server 2022.1 | 7.1 |
| Azure DevOps Server 2022 | 7.0 |
| Azure DevOps Server 2020 | 6.0 |
| Azure DevOps Server 2019 | 5.0 |
| TFS 2018 Update 2, Update 3 | 4.1 |
| TFS 2018 RTW, Update 1 | 4.0 |
| TFS 2017 Update 2 | 3.2 |
| TFS 2017 Update 1 | 3.1 |
| TFS 2017 RTW | 3.0 |
| TFS 2015 Update 3, Update 4 | 2.3 |
| TFS 2015 Update 2 | 2.2 |
| TFS 2015 Update 1 | 2.1 |
| TFS 2015 RTW | 2.0 |

"REST API versions are compatible with the Server version listed, as well as
Server versions that are newer" [S8]. 7.2 is listed only for "vNext"; which
released on-premises version supports it is not stated (unverified). Every
release up to and including Azure DevOps Server 2022.1 is therefore below the
default `7.2`, and rejects it as out of range.

**`7.2` may not work for Get Work Item on any server.** At 7.2 the resource is
documented only as `7.2-preview.3`, and only for Azure DevOps Services; the
on-premises views end at `azure-devops-server-rest-7.1` [S9]. A released
`7.2` for it is documented nowhere. By the negotiation rule in [S3], a
released version above the resource's released version needs `-preview`, so
`api-version=7.2` would be refused even by a server that supports 7.2.
Unverified, and the first thing to check against a live server.

**The request path and parameters.** `project` is an optional path segment
in every documented version from 4.1 to 7.2 [S9]. At API 1.0 (the TFS 2017
reference) the single work item route is documented without a project,
`{instance}/{collection}/_apis/wit/workitems/{id}`, and with `$expand` as its
only option; `fields` is documented only for the list route,
`workitems?ids=…` [S10]. Whether TFS 2017 and earlier accept the project
segment and `fields` on the single work item route: unverified.

**The response shape across versions.** Identity fields such as
`System.AssignedTo` and `System.CreatedBy` are strings,
`"Display Name <unique name>"`, at API 1.0 and 4.1, and objects with
`displayName` and `uniqueName` from 5.0 on [S9, S10]. Some identity fields
remain strings even at 7.x, e.g. `Microsoft.VSTS.Common.ActivatedBy` in the
7.1 and 7.2 samples [S9]. `internal/workitem` already accepts both forms: an
object becomes an identity, a string stays plain, and both print as
`Name <unique name>`. The markdown output is therefore unaffected by this
difference; a future `--json` output would expose it.

**No `api-version`.** In addition to the findings on `connectionData`: [S8]
words the rule as "should include" where [S1] says "must", and its own C#
sample sends `GET https://dev.azure.com/{organization}/_apis/projects`
without a version. Without a version the server answers at its latest
released version of the resource [S3, comment only], so the same request
yields the string identity shape from a TFS 2018 server and the object shape
from Azure DevOps Server 2019 and later. Server behaviour for GET without a
version: unverified, and for TFS 2017 and earlier there is not even SDK
evidence.

**Reading the supported version from an error.** The out-of-range message
names the latest version the server supports [S6], so a client could retry
with it. The message text is not a documented contract, and whether an
on-premises server installed in another language localises it is
unverified. `OPTIONS {url}/_apis` is the structured source of the same
information [S3].

**An empty value cannot be sent.** `internal/config/config.go` replaces an
empty `apiVersion` with the default, both after flags are applied and in
`LoadFile`.

**The alternatives:**

| Option | Server compatibility | Stability of the response shape | Works with no configuration |
|---|---|---|---|
| No `api-version` at all | Every server that accepts a GET without a version: TFS 2018 Update 2 and later by SDK evidence, earlier unverified | Varies by server; irrelevant to the markdown output, relevant to `--json` | Yes |
| No version by default; `--api-version`, `TFSCLI_API_VERSION`, and `apiVersion` as overrides | As above, and a user can still pin a version | Varies by default, pinned when configured | Yes |
| The lowest version all target servers support as the default, e.g. `1.0` | `1.0` is supported from TFS 2015 to Azure DevOps Server 2022 [S1]; a higher floor such as `4.1` excludes TFS 2017 and 2015 | Stable, at the oldest shape | Yes, if the request shape tfscli sends (project segment, `fields`) is accepted at that version, which is unverified for `1.0` |
| Negotiation through `OPTIONS {url}/_apis` before each call | Depends on the `OPTIONS` request being available and answered with the PAT; unverified before TFS 2018 | Varies by server | Yes, at the cost of a second request on every call, since tfscli keeps no cache |
| Retry with the version named in an out-of-range error | Every server that names its version in the error | Varies by server | Yes, at the cost of a second request on servers below the default, and only while the message text keeps its form |

### Decisions

The decisions below cover every request, not only the login verification:
`auth login` and the other commands share `apiclient` and the configuration
chain, so the API version cannot be settled for one without the other.

**1. No `api-version` by default.** When no version is configured, requests
carry no `api-version`, neither in the query nor in the `Accept` header. A
version set by `--api-version`, `TFSCLI_API_VERSION`, or `apiVersion` in the
config file is sent unchanged. The built-in default `7.2` is removed.

Reasons: a GET without a version works on every server that accepts it, and
Microsoft's own Node SDK and documentation samples rely on that; the user
does not need to know the server's API version; the login verification and
the commands behave alike, and `connectionData` is served at its latest
preview version without tfscli appending `-preview`; a user can still pin a
version.

Accepted costs: the documentation calls the version required; GET without a
version is unverified on TFS 2017 and earlier, which do not work without
configuration today either; the response shape depends on the server, which
the markdown output tolerates and which has to be settled together with
`--json`; a POST request, needed by future commands such as batch get or
WIQL, is refused without a version, so those commands need their own
approach when they arrive.

Rejected:
- The lowest version all target servers support, such as `1.0`: it is
  unverified whether the request tfscli sends (project segment, `fields`) is
  accepted at that version, and it fixes the oldest response shape on every
  server.
- Negotiation through `OPTIONS {url}/_apis`: a second request on every call,
  since tfscli keeps no cache, and unverified availability before TFS 2018.
- Retrying with the version named in an out-of-range error: it depends on
  message text that is not a contract and may be localised, and costs a
  second request on every older server.
- Keeping `7.2`: by the findings above it is likely refused by every server
  for Get Work Item.

**2. The login verification never carries `api-version`.** The
`connectionData` request made by `auth login` is sent without a version,
whatever the flag, the environment, or the config file say. The appending of
`-preview` in `internal/cli/auth.go` is removed. `--api-version` given to
`auth login` is refused with a `config` error rather than silently ignored;
`TFSCLI_API_VERSION` and `apiVersion` are ignored by `auth login`, since they
are set for the other commands, and its help says so.

Reasons: `connectionData` has no released version, so a version pinned for
the commands, such as `6.0`, would be refused for it unless `-preview` were
appended, which rests on an unverified assumption; the version a user pins
fixes the shape of data, which the verification of the URL and PAT pair does
not read; an out-of-range error cannot occur during login, so login needs no
next step for it; the special case "login reads the version from the config
file only" disappears.

Accepted cost: a server that refuses `connectionData` without a version, if
one exists (TFS 2017 and earlier are unverified), leaves no way to log in.
Such a server cannot be logged in to today without a config file either; it
is addressed when one is reported.

Rejected:
- The login verification reads the version from the same chain as the
  commands: a pinned released version is refused for a preview resource, so
  the `-preview` workaround would have to stay.
- Keeping the version from the config file only, as now: the same problem,
  plus an exception to the configuration chain that has no reason.
- Trying without a version first and with one second: no server is known to
  need the second attempt, and the version for it would have to be guessed.
- Replacing the verification request: no candidate identifies the user, and
  `projects` needs a collection, which login does not have (see the findings
  above).

**3. A refused API version names its source and the next step.** After
decisions 1 and 2 the server can refuse a version only for a command, and
only when the user configured one. The error is recognised by the exception
name in the TFS error body (`typeKey`, alongside `message`), not by the
message text. When it names a version error and a version was configured,
the message from the server is kept and followed by the source of the
version (`--api-version`, `TFSCLI_API_VERSION`, or `apiVersion` in the config
file with its path) and the next step: remove the setting to let the server
choose, or lower it. Every other error is unchanged.

Reasons: the exception name does not depend on the server's language or on
the wording of the message; the server's message is kept, and it names the
latest version the server supports; an error about an unknown field in
`--fields`, which is also HTTP 400, does not receive a misleading hint.

Accepted costs: the `typeKey` values are known from memory only —
`VssVersionOutOfRangeException` for a version above the server's, and
probably a separate one for a preview resource requested without `-preview`
— and are unverified; until they are confirmed the hint may not appear, which
leaves the user with the server's message, as today. The configuration has to
record where the version came from.

Rejected:
- Matching the message text, such as "out of range": it breaks on a server
  installed in another language.
- Adding the hint to every HTTP 400 when a version is configured: it
  misleads on an error about `--fields`, which contradicts the rule not to
  guess.

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
**Status:** Done

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
**Status:** Done

### TASK-04 `no-default-version`
**Description:** Implement decisions 1 and 2 together, since removing the
default alone would leave `auth login` sending `api-version=-preview`.
- Remove `DefaultAPIVersion` and the substitution of an empty `apiVersion`
  in `internal/config/config.go`, and make `internal/apiclient/apiclient.go`
  add `api-version` only when a version is configured.
- In `internal/cli/auth.go`, send the `connectionData` request without a
  version and remove the appending of `-preview`; refuse `--api-version` on
  `auth login` with a `config` error; state in the command's help that the
  API version settings do not apply to it.
- Update the `--api-version` help text in `internal/cli/cli.go`; the
  configuration tables, the request shown, the verbose example, and the
  description of `auth login` in `README.md` and `skills/tfscli/SKILL.md`;
  `config.example.json`; the target API version and the "Authentication"
  section in `docs/project-brief.md`; the `CLAUDE.md` constraint that names
  `7.2` as the default; and `docs/architecture.md` where it names the
  default.
- Record the change in `CHANGELOG.md` through the `changelog` skill.
**Definition of done:** With no version configured, a request carries no
`api-version` parameter; a version from the flag, the environment, or the
config file is sent unchanged. The login verification carries no
`api-version` whatever is configured, and `auth login --api-version …` fails
with a `config` error. Tests cover each case. No document outside
`docs/archive/`, earlier `CHANGELOG.md` releases, and the Context section of
this file describes `7.2` as the default or says that `auth login` reads the
API version. `make check` passes.
**Status:** Pending

### TASK-05 `version-error-step`
**Description:** Implement decision 3. Record in `internal/config` which
source supplied the API version: the flag, the environment, or the config
file with its path. In `internal/apiclient/apiclient.go`, read `typeKey` from
the TFS error body next to `message`; when it names a version error and a
version was configured, append the source and the next step to the server's
message. Treat `VssVersionOutOfRangeException` as a version error, and the
exception for a preview resource requested without `-preview` once its name
is known; keep the set of names in one place so that the live check can
correct it. Update the `config` row in the error table of
`skills/tfscli/SKILL.md` and its representative messages. Record the change
in `CHANGELOG.md` through the `changelog` skill.
**Definition of done:** Against a stub server returning a version error with
the expected `typeKey`, the message names the source of the version and the
next step for each of the three sources. A 400 with another `typeKey`, or
with none, keeps its current message. Tests cover these cases. `make check`
passes.
**Status:** Pending

### TASK-06 `live-check`
**Description:** Against a live TFS or Azure DevOps Server, preferably more
than one release, check the assumptions the decisions rest on: that
`_apis/connectionData` and `{project}/_apis/wit/workitems/{id}`, with and
without `fields`, are answered without `api-version`; that
`api-version=7.2` is refused for Get Work Item; the `typeKey` values of the
out-of-range error and of a preview resource requested without `-preview`;
how `connectionData` answers a URL that contains a collection, including
`instanceId` and `deploymentId`; and how an anonymous request and an invalid
PAT are answered (TD-01 in `docs/tech-debt.md`).
**Definition of done:** Each assumption is recorded in the Context section as
confirmed or refuted, with the server release it was checked on. A refuted
assumption is followed by a new task in this file, or by a new tasks file
when it changes a decision.
**Status:** Pending
