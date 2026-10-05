---
name: changelog
description: Maintain CHANGELOG.md for users, not developers — Keep a Changelog format, user-visible changes only, the six standard sections, strict entry format, and the release procedure.
argument-hint: "[change to record | release X.Y.Z]"
---

Format and rules for maintaining `CHANGELOG.md`. Use this skill when adding an entry,
reviewing whether a change belongs in the changelog, or preparing a release section.

## Project version first

If the current repository contains `.claude/skills/changelog/SKILL.md` and that file is
not the one you are reading, read it and follow it instead of this file.

Where the project's own documentation (for example `CONTRIBUTING.md` or `CLAUDE.md`)
sets a different convention, the project's convention wins over this skill.

## Existing changelogs

If `CHANGELOG.md` already exists in a format other than the one described here, do not
convert it unless the user asks for that explicitly. Add new entries in the format the
file already uses, and offer the conversion as a separate change.

## Audience

Every entry is written for the **user**, not the developer. The changelog describes
changes in observable behaviour — not implementation details, refactoring, or test
coverage. If a change is invisible to the user, it does not belong here.

## Structure

The file follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/), with one
deliberate omission: no version comparison links are kept at the end of the file. They
are addressed to developers rather than users, and every release would have to rewrite
them.

The file opens with a short preamble naming the format, followed by one section per
release, ordered from newest to oldest. The top section is always `[Unreleased]` and
collects changes that have not yet been released. Sections are separated by their
headings alone — no horizontal rules between them.

```markdown
# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- ...

## [0.2.0] - 2026-03-15

### Changed
- ...

### Fixed
- ...

## [0.1.0] - 2026-02-28

### Added
- ...
```

If the project declares a versioning scheme — in its contributing guide or similar
documentation — the preamble names it as well, for example: "and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)." Do not name a scheme the
project has not declared.

A version heading is `## [X.Y.Z] - YYYY-MM-DD`: the version in brackets, a hyphen
surrounded by spaces, and the release date in ISO 8601 format.

## Sections

Use exactly these six section names, in this order. A section heading appears only
when it has at least one entry — in `[Unreleased]` as well as in released versions.
When `[Unreleased]` has no entries, it consists of its heading alone. The first entry
for a section adds that section's heading at its position in the order below.

| Section | Use for |
|---------|---------|
| `Added` | New features and capabilities visible to the user. |
| `Changed` | Existing behaviour that has been intentionally altered. |
| `Deprecated` | Features or options that still work but will be removed in a future release. |
| `Removed` | Features or options that no longer exist. |
| `Fixed` | Bugs that were observable by the user and are now resolved. |
| `Security` | Fixed vulnerabilities, so that users know to upgrade. |

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

## What to include and what to skip

| Include | Skip |
|---------|------|
| New feature visible to the user | Refactoring without behaviour change |
| Bug fix the user could observe | Dependency updates with no visible effect |
| Config field or CLI flag change | New or updated tests |
| Change in existing feature behaviour | Documentation edits |
| Removed or deprecated feature or option | Internal code cleanup |

## Releasing

If the project automates the release, the automation performs these steps; do not
perform them by hand. Otherwise, before tagging a release:

1. Insert a version heading `## [X.Y.Z] - YYYY-MM-DD` directly below `## [Unreleased]`,
   so that the entries collected so far belong to the new version and `[Unreleased]`
   is left empty.
2. Commit the result before tagging.

```markdown
## [Unreleased]

## [0.2.0] - 2026-03-15

### Fixed
- ...
```
