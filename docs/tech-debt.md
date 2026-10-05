## tech-debt.md

### Active

#### TD-07 `main-not-protected`
**Problem:** `main` has no server-side protection. The rule that changes reach it
only through pull requests with a green `check` is a convention that GitHub does not
enforce.
**Why deferred:** The repository is private on the free plan, where GitHub does not
enforce rulesets. Protection becomes possible only when the repository is made
public or the account moves to GitHub Pro, and neither has been decided.
**Trigger:** The repository becomes public, or the account moves to GitHub Pro. The
work found necessary at that point:
- A branch ruleset on the default branch: pull request required with 0 approvals,
  the `check` status check from GitHub Actions required with branches up to date,
  linear history, deletions and force pushes blocked; only rebase merging allowed
  in the repository settings, and branches updated with "Update with rebase".
- The `Release` workflow cannot push with `GITHUB_TOKEN` past such a ruleset, since
  `github-actions[bot]` cannot be a bypass actor. It pushes with a deploy key that
  has write access and is the ruleset's only bypass actor; the private key is a
  secret of an environment `release` limited to `main`, so a workflow on another
  branch cannot read it, and reaches `actions/checkout` through `ssh-key`.
- A push made with a deploy key starts workflows, unlike one made with
  `GITHUB_TOKEN`: the comment on the push step in `release.yml` must change, and the
  `push: main` trigger in `ci.yml` is removed, since with up-to-date branches
  required it checks nothing a pull request has not and would only rerun CI on every
  release commit.
- The ruleset is enabled after the `Release` change is merged, and `CONTRIBUTING.md`
  ("Commits and branches") and `CLAUDE.md` ("Git workflow") again describe `main` as
  protected.
**Added:** 2026-10-05, in conversation; replaces TASK-01 of
`docs/archive/2026-10-04-tasks-release-bypass.md`

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
