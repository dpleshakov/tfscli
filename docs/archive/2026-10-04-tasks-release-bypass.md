# 2026-10-04-tasks-release-bypass.md

**Status:** Archived

---

### TASK-01 `release-push-bypass`
**Description:** Make the `Release` workflow able to push the release commit and the tag to `main` once the branch is protected by a ruleset that requires pull requests. The approach is to be chosen when the task is taken up.
**Definition of done:** With the ruleset enabled, the `Release` workflow pushes the release commit and the tag to `main` without a pull request.
**Status:** Skipped — the repository is private on the free plan, where GitHub does not enforce rulesets, so there is nothing to bypass; deferred to TD-07 in `docs/tech-debt.md`
