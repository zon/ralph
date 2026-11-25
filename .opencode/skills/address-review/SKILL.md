---
name: address-review
description: Addresses one review comment from a review.html report and marks it fixed in the report. Selects comments at medium severity or higher by default. The user can adjust the minimum severity. Deletes the report when nothing remains to address. Use when the user asks to address, fix, or work through a finding or comment from review.html or a code review report.
---

# Address Review

Fix one review comment per run, so the change stays reviewable. The report is an artifact. The fix belongs in the source.

## Definitions

- **Comment** — one review comment in the report. Its `mark_id` (a UUID plus a sequence suffix such as `...#0`) identifies it.
- **Marked** — the comment's state is `fixed` or `ignored`. The list hides marked comments by default.

## Steps

1. **Locate the report.** Use `review.html` at the repository root unless the user names another path. If it is missing, there is nothing to address.

2. **List the comments.** The file is large, so use the CLI rather than reading it whole. `zpecs review list` prints JSON with a `comments` array and a `summary` object, and lists only open comments by default. The minimum severity defaults to `medium`, matching step 3:

   ```bash
   zpecs review list --severity medium review.html
   ```

   Each comment has `mark`, `severity`, `category`, `path`, `lines`, `mark_id`, and `body`. The mark is `fixed`, `ignored`, or `open`. `body` is an excerpt for open comments and empty for marked ones. `summary` counts the listed comments, so with the default open filter it reports how many open comments remain. Lower the filter to `low` when the user names a comment below the minimum or asks to include `low` comments. Pass `--mark all` to include fixed and ignored comments.

3. **Choose one comment.** The minimum severity defaults to `medium` and matches the list filter. Take `critical`, `high`, and `medium` comments and ignore `low`. Raise the minimum when the user names a higher severity, or lower it when they ask to include `low`. Use the comment the user names, even below the minimum. Otherwise choose by severity, then category (`bug` and `security` before `performance`, `maintainability`, `documentation`, and `style`), then a self-contained change you can verify locally. Skip comments that need a redesign, a live cluster, or credentials, and say why.

4. **Read the comment.** Run `zpecs review get` with the mark id and read its `body`, `existing_code`, and `suggestion`. Record the mark id, source path, and line range.

   ```bash
   zpecs review get 'cb245837-1736-4e33-b1f3-84036c32e6d5#2' review.html
   ```

5. **Verify against the code.** Find the code the comment describes and compare it with the current source, because the `existing_code` snippet can be stale. If the code already has the fix, report the stale comment and return to step 3. A suggested change is a proposal, not a patch. Adapt it to the surrounding code and the repository's conventions.

6. **Implement the fix.** Make the smallest change that resolves the comment. Follow the repository's instructions and prose rules. Do not fix other comments.

7. **Bump the version** when the repository tracks a version for its output. Bump once per branch, not once per comment. Find the documented version source and its bump levels, then apply the level the branch's change calls for. Skip the bump when the branch already has one covering its work or the repository has no version.

8. **Verify.** Read the testing rules, such as `docs/testing.md`. Test the affected package and build the whole project. Add or adjust tests when the comment describes untested or wrongly tested behavior. Report a pre-existing failure as unrelated and show why.

9. **Mark the comment fixed.** Run `zpecs review fix` with the report path and the full `mark_id`. It records the comment as fixed and leaves the rest of the report untouched. It stops when the id matches no comment or several comments, so pass the full id rather than the snippet or file path, because several comments can share a file.

   ```bash
   zpecs review fix 'cb245837-1736-4e33-b1f3-84036c32e6d5#2' review.html
   ```

   Do not regenerate the report. The list hides marked comments by default.

10. **Delete the report once no unmarked comment with severity `medium` or higher remains.** Rerun `zpecs review list --severity medium`, or `--severity low` when the user included `low`, after the mark is written and read the summary object. Keep the report when the user asked to include `low` and unmarked `low` comments remain.

11. **Report** the mark id, severity, category, and source path. Also report what changed, the version bump or why none was needed, the verification result, whether you marked the comment fixed, and whether you deleted the report.

## Blocking behavior

- **No report** — say there is nothing to address and stop.
- **No unmarked comment with severity `medium` or higher remains** — delete the report and stop. Keep it if the user included `low` and unmarked `low` comments remain.
- **Verification cannot run** — name the command that could not run and why, and report the comment as addressed but unverified.
