# Changelog Guide

This document describes the format and rules for maintaining `CHANGELOG.md`.

---

## Audience

Every entry is written for the **user**, not the developer. The changelog describes
changes in observable behaviour — not implementation details, refactoring, or test
coverage. If a change is invisible to the user, it does not belong here.

---

## Structure

The file contains one section per release, ordered from newest to oldest.
The top section is always `[Unreleased]` and collects changes that have not yet been tagged.

```
## [Unreleased]

### Added
### Fixed
### Changed
### Removed

---

## [0.2.0] — 2026-03-15

### Fixed
- ...

---

## [0.1.0] — 2026-02-28

### Added
- ...
```

---

## Sections

Use exactly these four section names. Use only the sections that have entries
— omit empty ones.

| Section | Use for |
|---------|---------|
| `Added` | New features and capabilities visible to the user. |
| `Fixed` | Bugs that were observable by the user and are now resolved. |
| `Changed` | Existing behaviour that has been intentionally altered. |
| `Removed` | Features or options that no longer exist. |

---

## Entry format

One entry per line, starting with `-`. One sentence, ending with a period.
No issue numbers, no commit hashes, no author names.

**Write what changed in behaviour, not what changed in code.**

Start each entry with the subject of the change, not a verb:

```
✓  Export button now produces a file named after the current date.
✓  Filters are no longer reset after a manual data refresh.

✗  Added date-based filename generation to the export button.
✗  Fixed filter state preservation on refresh.
```

---

## What to include and what to skip

| Include | Skip |
|---------|------|
| New feature visible to the user | Refactoring without behaviour change |
| Bug fix the user could observe | Dependency updates with no visible effect |
| Config field or CLI flag change | New or updated tests |
| Change in existing feature behaviour | Documentation edits |
| Removed feature or option | Internal code cleanup |

---

## Releasing

Before tagging a release:

1. Rename `[Unreleased]` to `[X.Y.Z] — YYYY-MM-DD`.
2. Add a new empty `[Unreleased]` section above it.
3. Commit the result before tagging.

```markdown
## [Unreleased]

### Added
### Fixed
### Changed
### Removed

---

## [0.2.0] — 2026-03-15

### Fixed
- ...
```
