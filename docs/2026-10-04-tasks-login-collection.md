# 2026-10-04-tasks-login-collection.md

**Status:** Active

## Context

The first live run of `tfscli auth login` failed with
`Error [auth]: PAT is invalid or expired (HTTP 401)` although the token was
valid. Checks with curl against the same server established:

- `GET {url}/_apis/connectionData` (server level) answers 401 with an empty
  body and an `ActivityId` header, for a token with full access as well.
- `GET {url}/{collection}/_apis/connectionData` with the same token answers
  200 and carries `authenticatedUser.id` and `providerDisplayName`.
- Work item requests at collection level with the same token answer 200.

The Microsoft documentation for Azure DevOps Server describes creating a PAT
as "Enter a name, select the collection, and set an expiration date"; for TFS
2018, "If you have more than one organization, you can also select the
organization where you want to use the token"
(https://learn.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate).
A PAT is therefore issued for a collection, the on-premises counterpart of an
organization. Whether on-premises servers also offer a token for all
collections is not documented.

Decisions:

- The collection becomes part of the credential, next to the URL and the PAT:
  `auth.json` and `TFSCLI_AUTH` carry a required `collection` field. The
  `--collection` flag, the `TFSCLI_COLLECTION` variable, and the `collection`
  config key are removed. There are no users yet, so no migration is provided.
- `auth login` asks for the URL, the collection, and the token, in that order.
  The collection has no default; an empty one is a `config` error.
- The login check moves to `{url}/{collection}/_apis/connectionData`. This
  does not narrow server compatibility: the resource is available at
  collection level wherever it is at server level.
- When the last segment of the URL path equals the collection, compared
  case-insensitively, it is removed once and a notice says so. The success
  line always names the stored URL and collection.
- A 404 on the login check is reported as `not_found` naming the combined URL
  and asking to check the server URL and the collection name; it does not
  claim which of the two is wrong. A server-supplied `message`, if any, is
  kept.
- The 401 message states first that the server did not accept the PAT, then
  the possible causes, including IIS Basic Authentication, which the
  Microsoft documentation names as preventing PAT authentication on Azure
  DevOps Server.
- Further diagnostics (response headers under `--verbose`, `ActivityId` in
  messages) are out of scope.

This resolves the open question "tells server URL from collection URL" in
`docs/archive/2026-10-01-tasks-login-api-version.md`: a URL that already
contains the collection is either corrected by the stripping rule or yields a
404 on the doubled path. TD-01 is not affected.

---

### TASK-01 `collection-in-credential`
**Description:** Add a required `collection` field to `config.Auth`, validated
in `parseAuth` for both `auth.json` and `TFSCLI_AUTH`. Fill
`Config.Collection` only from the credential; remove the `collection` config
key, `TFSCLI_COLLECTION`, the `--collection` flag, `Overrides.Collection`,
and the "collection is not set" check in `config.Load`. In `auth login`,
prompt for the collection between the URL and the token, verify against
`{url}/{collection}/_apis/connectionData`, store all three values, and print
`Logged in to <url>, collection <name>, as <user>`.
**Definition of done:** `make check` passes; tests cover a credential without
`collection` (from the file and from `TFSCLI_AUTH`), an empty collection at
the login prompt, the collection-level verification path, the stored
`auth.json` content, and the success line; `--collection` is rejected as an
unknown flag.
**Status:** Done

### TASK-02 `login-url-collection-strip`
**Description:** In `auth login`, when the last path segment of the
normalised URL equals the entered collection (case-insensitive), remove that
segment once before verification and print a notice that the collection was
removed from the URL.
**Definition of done:** `make check` passes; tests cover a URL ending in the
collection with different letter case, a URL not ending in it, and a URL
whose path consists of the collection only.
**Status:** Done

### TASK-03 `login-not-found-message`
**Description:** In `auth login`, map a 404 from the verification request to
a `not_found` error naming the combined URL and asking to check the server
URL and the collection name, keeping the server-supplied message when the
response carries one.
**Definition of done:** `make check` passes; tests cover a 404 with an empty
body and a 404 with a TFS error JSON body, asserting the full stderr line.
**Status:** Done

### TASK-04 `auth-error-wording`
**Description:** Change the generic 401 message in `categoryFor` to "the
server did not accept the PAT (it may be invalid, expired, or revoked; if it
is valid, IIS Basic Authentication may be enabled on the server, which only
an administrator can turn off)". Update the tests that assert the old text
and the `auth` row of the error table in `README.md`.
**Definition of done:** `make check` passes; no occurrence of "PAT is invalid
or expired" remains outside `CHANGELOG.md` and `docs/archive/`.
**Status:** Pending

### TASK-05 `contract-docs`
**Description:** Update the documents that describe the configuration and
credential contract: `docs/project-brief.md` (minimal configuration, config
example, overridable variables and flags), `CLAUDE.md` (credential and config
precedence constraints), `README.md` (settings table, `auth login`,
`TFSCLI_AUTH` format, examples), `docs/architecture.md` (config flow and
`Config` description), and `docs/release-footer.md`.
**Definition of done:** no document outside `CHANGELOG.md` and
`docs/archive/` mentions `--collection`, `TFSCLI_COLLECTION`, or a
`collection` config key; every description of the credential names the URL,
the collection, and the PAT.
**Status:** Pending

### TASK-06 `changelog`
**Description:** Add entries to `CHANGELOG.md` following the `changelog`
process: Removed (the flag, the variable, the config key), Changed (the
credential carries the collection; `auth login` asks for it; the 401
message), Fixed (`auth login` on servers that do not accept the PAT at server
level).
**Definition of done:** the entries are present under the unreleased section
in the format the `changelog` process prescribes; `make check` passes.
**Status:** Pending

### TASK-07 `live-check`
**Description:** Run `tfscli auth login` and `tfscli workitem get` against
the live TFS instance on which the original failure occurred. Performed by
the user.
**Definition of done:** `auth login` stores the credential and prints the
success line; `workitem get` returns a work item using the stored
credential.
**Status:** Pending
