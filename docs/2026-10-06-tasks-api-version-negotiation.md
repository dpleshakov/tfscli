# 2026-10-06-tasks-api-version-negotiation.md

**Status:** Active

## Context

Goal: every command works on any server without the user setting an API
version. Today `wit work-items get-batch` and `wit wiql query-by-wiql` fail
with "No api-version was supplied" on servers that require a version for
POST, and neither the user nor an agent can be expected to know the right
value.

Microsoft documents `api-version` as required on every request; in practice
GET requests work without it, while POST requests are refused by some
servers. The official SDKs (.NET, Go, Node, Python) discover versions with
`OPTIONS {url}/{collection}/_apis`, which lists every resource with its
`minVersion`, `maxVersion`, `releasedVersion`, and `resourceVersion`
(`getResourceLocationsFromServer` in `azuredevops/client.go` and
`negotiateRequestVersion` in `azuredevops/versionnegotiation.go` of
`microsoft/azure-devops-go-api`; `client.py` of
`microsoft/azure-devops-python-api`).

Decisions already made:

- When no version is configured, every request except the verification in
  `auth login` is preceded by `OPTIONS {url}/{collection}/_apis` and carries
  the `releasedVersion` of its resource — the version the server chooses for a
  request without one, so the output does not change. A resource without a
  released version (`0.0`) gets `{maxVersion}-preview.{resourceVersion}`.
- The response is not cached: an on-disk cache is excluded by the product
  constraints, and an in-process cache gains nothing in a CLI. Each invocation
  makes one extra request.
- A configured version (config file, `TFSCLI_API_VERSION`, `--api-version`) is
  sent as is, without `OPTIONS`.
- Negotiation never fails a command on its own. A 401 or a transport error on
  `OPTIONS` is reported at once in its usual category, since the main request
  would fail the same way. Any other failure — 400, 403, 404, 405, 5xx, an
  unparseable response, the resource missing from the list — falls back to
  sending the request without a version, as before, so that no server that
  works today stops working.
- Under `--verbose` the `OPTIONS` request is logged like any other, and a
  fallback adds one line saying that the version was not negotiated and why.
  Nothing is printed without `--verbose`.
- When a request sent without a version after a fallback is refused for lack
  of one, the `config` error says that negotiation failed and why, names
  `apiVersion` with the config file path, `TFSCLI_API_VERSION`, and
  `--api-version`, and says that the value must not exceed the server's
  version. This replaces TASK-08 of
  `docs/2026-10-04-tasks-first-run-errors.md`.
- No new error category.
- The constraint "requests carry `api-version` only when one is configured" in
  `docs/project-brief.md` and `CLAUDE.md`, and the "REST API version" section
  of `README.md`, change together with this work.

### Findings (TASK-01)

Sources: `microsoft/azure-devops-go-api`, branch `dev`, files
`azuredevops/v7/client.go` [G1], `azuredevops/v7/models.go` [G2],
`azuredevops/v7/connection.go` [G3], `azuredevops/v7/versionnegotiation.go`
[G4], `azuredevops/v7/workitemtracking/client.go` [G5]; the error response
quoted in https://dexterposh.github.io/posts/002-azdo-tip-api/ [D1].

- **Request.** `getResourceLocationsFromServer` sends `OPTIONS` to the client's
  base URL joined with `_apis`, with `Accept: application/json` and no
  `api-version` [G1]. For an on-premises server the base URL is the
  collection URL: `GetClientByResourceAreaId` falls back to the connection
  URL because "resourceAreaInfo will be nil for on prem servers" [G3]. tfscli
  therefore sends `OPTIONS {url}/{collection}/_apis`.
- **Response.** A collection wrapper `{"count": n, "value": [...]}`, read with
  `UnmarshalCollectionBody` [G1]. Each element is an `ApiResourceLocation`
  [G2]: `id` (GUID), `area`, `resourceName`, `routeTemplate`, `minVersion`,
  `maxVersion`, `releasedVersion` (strings such as `"7.1"`), and
  `resourceVersion` (an integer). `releasedVersion` is "the latest version of
  this resource location that is in Release (non-preview) mode"; the
  negotiation treats `0.0` as "not released" [G4].
- **Location ids** [G5]: Get Work Item and Work Items - List share
  `72c7ddf8-2cdc-4f60-90cd-ab71c14a399b`; Get Work Items Batch is
  `908509b6-4248-4475-a1cd-829139ba419f`; Query By Wiql, with and without a
  team, is `1a9c53f7-f243-4447-b110-35ef023636e4`.
- **Matching: by location id.** The SDK indexes the response by `id` and
  fails with `LocationIdNotRegisteredError` when the id is absent [G1]. The
  id is what the server publishes as the identity of a location and is the
  same in every release. Matching by area and resource name is rejected: the
  pair is not documented as unique, and the work items resource already has
  two locations — `72c7ddf8-…` and `62d3d110-0047-428c-ad3c-4fe872c91c74`,
  used for Create Work Item and Get Work Item Template [G5] — whose resource
  names have not been checked; an ambiguous match would pick a version for
  the wrong route.
- **Where the version goes.** The SDK sends it in the `Accept` header
  (`application/json;api-version=…`) [G1]; the server error [D1] accepts it
  "either as part of the Accept header … or as a query parameter". tfscli
  keeps the query parameter, so that the version stays visible in the URL
  logged under `--verbose`.
- **Refusal without a version** [D1]: `typeKey` `VssVersionNotSpecifiedException`,
  message `No api-version was supplied for the "POST" request. …`. The
  report is from Azure DevOps Services; that an on-premises server returns
  the same `typeKey` is unverified.
- **TFS 2015 and 2017.** No source was found that shows the `OPTIONS`
  response of these releases. Unverified; the fallback to a request without
  a version covers a server that does not answer it.

---

### TASK-01 `options-research`
**Description:** From the SDK sources and Microsoft documentation, establish:
the shape of the `OPTIONS {url}/{collection}/_apis` response and the fields
of an `ApiResourceLocation`; the location ids of the resources tfscli calls —
Work Items (Get Work Item, Work Items - List), Work Items Batch, and Wiql;
whether a request is matched to its entry by location id, as the SDKs do, or
by area and resource name, and the reasons; what is known about `OPTIONS` on
TFS 2015 and 2017; and the `typeKey` of the error a server returns for a
request that carries no `api-version`.
**Definition of done:** The findings are recorded in the Context section,
each with its source; anything not confirmed is marked as unverified; the
matching method is decided and recorded with the rejected alternative.
**Status:** Done

### TASK-02 `version-selection`
**Description:** In `internal/apiclient`, parse the `OPTIONS` response and add
a pure function that chooses the version for a resource from its entry: the
`releasedVersion`, or `{maxVersion}-preview.{resourceVersion}` when the
released version is `0.0`. A response that cannot be parsed, and a resource
missing from it, are reported to the caller as distinct outcomes, so that the
fallback can say why it happened.
**Definition of done:** Unit tests cover a released resource, a
preview-only resource, a resource missing from the list, and an unparseable
response. `make check` passes.
**Status:** Pending

### TASK-03 `client-negotiation`
**Description:** Use the version selection in `Client`. Callers name the
resource of each request, which changes `Get` and `Post` and the `APIClient`
interfaces in `internal/workitem` and `internal/wiql`. A configured version
is sent without `OPTIONS`. A 401 or a transport error on `OPTIONS` is
returned as the error of the command; any other failure falls back to a
request without a version and logs one line under `--verbose` saying why,
through a `Logger` method suited to it rather than `Warn`. The verification
in `auth login` (`verify` in `internal/cli/auth.go`) makes no `OPTIONS`
request.
**Definition of done:** Tests against a stub server cover: a negotiated
version on GET and on POST; a configured version sent without `OPTIONS`; a
401 and a transport failure on `OPTIONS`; each fallback case with its
`--verbose` line; and `auth login` sending no `OPTIONS`. `make check` passes.
**Status:** Pending

### TASK-04 `missing-version-error`
**Description:** When a request sent without a version after a fallback is
refused for lack of one, recognise the refusal by the `typeKey` established
in TASK-01, falling back to the message text only if no `typeKey` is
available, and add the next step to the `config` error: that the version
could not be negotiated and why; `apiVersion` in the config file with the
path of the file, `TFSCLI_API_VERSION`, and `--api-version`; and that the
value must not exceed the highest version the server supports, as listed in
the "REST API version" section of `README.md`.
**Definition of done:** Tests cover the refusal after a fallback; a refusal
of a configured version keeps its current message. `make check` passes.
**Status:** Pending

### TASK-05 `docs`
**Description:** Bring the documents in line with the new behaviour: the
constraint on `api-version` in `docs/project-brief.md` (Constraints, Design
Principles, Configuration) and `CLAUDE.md`; the description of
`internal/apiclient` and the data flow in `docs/architecture.md`; the "REST
API version" and "Request logging" sections of `README.md`, and the
`config` row of its error table; `skills/tfscli/SKILL.md` wherever it speaks
of the API version; and an entry in `CHANGELOG.md` per the `changelog`
skill.
**Definition of done:** No document describes requests as carrying
`api-version` only when one is configured; the README describes the
`OPTIONS` request and the fallback; the changelog entry is in place. `make
check` passes.
**Status:** Pending
