## tech-debt.md

### Active

---

### Closed

#### TD-01 `login-anonymous-detection`
**Fixed:** 2026-10-05 — dropped without a change: no server with anonymous
access is available to verify against; recorded as a known limitation in
`README.md` until a user reports such a server

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
