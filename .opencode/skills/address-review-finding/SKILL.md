---
name: address-review-finding
description: Selects one finding from a review.html code review report, fixes it in the codebase, and marks it fixed in the file. Ignores low findings and deletes the report when only low findings remain. Use when the user asks to address, fix, or work through an issue or finding from review.html or a code review report.
---

# Address Review Finding

Address exactly one finding per run, so the change stays small enough to review. The report is an artifact; the fix belongs in the source it describes.

## Definitions

- **Finding** - one `<div class="comment-card">` in the report. It carries a severity, a category, and a mark id.
- **Mark id** - the value of `data-mark-id`, a UUID plus a sequence suffix such as `178217b6-7cc9-4769-8454-936bb9639d9e#0`. It identifies the finding on disk and in browser storage.
- **Marked** - the card carries `data-mark="fixed"` or `data-mark="ignored"`, or its id has an entry in the viewer's `localStorage`.
- **Existing code** - the snippet under `Existing Code`, captured when the report was generated. Line numbers drift, so verify it against the current source before editing.

## Steps

1. **Locate the report.** Use `review.html` at the repository root unless the user names another path. Confirm it is a session detail viewer by checking for `data-comment-card`. A missing report is not a problem: it means there is nothing to address. Report that and stop.

2. **List the findings.** The file is large, so grep or script it rather than reading it whole. This prints the addressable findings with their file and line range:

   ```bash
   python3 - <<'PY'
   import re, html
   s = open('review.html', encoding='utf-8').read()
   files = [(m.start(), html.unescape(m.group(1))) for m in re.finditer(r'<span class="file-path">(.*?)</span>', s)]
   cards = re.finditer(r'<div class="comment-card"([^>]*)>(.*?)<div class="comment-actions">', s, re.S)
   def attr(attrs, name):
       m = re.search(name + r'="([^"]*)"', attrs)
       return m.group(1) if m else None
   def path_at(pos):
       path = None
       for p, f in files:
           if p >= pos:
               break
           path = f
       return path
   for m in cards:
       attrs, body = m.groups()
       if attr(attrs, 'data-mark'):
           continue
       lines = re.search(r'<span class="comment-lines sr-only">([^<]*)</span>', body)
       content = re.search(r'<div class="comment-content">(.*?)</div>', body, re.S)
       text = html.unescape(re.sub('<[^>]+>', '', content.group(1))).strip() if content else ''
       print(f"{attr(attrs, 'data-severity')}\t{attr(attrs, 'data-category')}\t"
             f"{path_at(m.start())}\t{lines.group(1) if lines else '?'}\t{attr(attrs, 'data-mark-id')}")
       print('   ', text[:300].replace('\n', ' '))
   PY
   ```

3. **Choose one finding.** Ignore `low` findings. If no other unmarked finding remains, delete the report and stop. Otherwise honor the finding the user names. If none is named, order candidates by severity (`critical`, `high`, `medium`), then prefer `bug` and `security` over `performance`, `maintainability`, `documentation`, and `style`, then prefer a self-contained change that can be verified locally. Skip findings that need a redesign, a live cluster, or credentials, and say why when you report.

4. **Read the finding in full.** Extract the complete `comment-content`, the `Existing Code` panel, and any `Suggested Change` panel. Record the mark id, source path, and line range.

5. **Verify against the code.** Open the named file and locate the code the finding describes. Treat the report as stale until the code matches. If the code already implements the fix, go back to step 3 for the next finding and report the stale finding. A suggested change is a proposal, not a patch: adapt it to the surrounding code and the repository's conventions.

6. **Implement the fix.** Make the smallest change that resolves the finding. Follow the repository's own instructions, including its testing rules and prose rules. Do not fix other findings in the same change.

7. **Bump the version.** Most repositories that version their output require a bump for every branch that changes it, and the branch may already carry changes without one. Find the version source the repository documents, such as `internal/version/VERSION` or a version constant, and read its bump levels. Bump once per branch, not once per finding: if the newest bump predates the branch's work, apply the level the branch's overall change calls for, usually a patch bump for a bug fix, a minor bump for a new feature, and a major bump for a breaking change. Leave the version alone when the branch already has a bump covering its work or when the repository does not version.

8. **Verify.** Read the repository's testing rules first, such as `docs/testing.md`, then run the tests for the affected package and the build wider than the change. Add or adjust tests when the finding describes untested or wrongly tested behavior. Report any failure that predates the change as unrelated and show why.

9. **Mark the finding fixed in the file.** Edit only the matching card. Do not regenerate the report.

   The card begins:

   ```html
   <div class="comment-card" data-comment-card data-category="bug" data-severity="high" data-mark-id="UUID#0" hidden>
   ```

   Add `data-mark="fixed"` to that tag and replace the hidden chip in the same card's `comment-meta`:

   ```html
   <span class="comment-badge mark-chip" data-mark-chip hidden></span>
   ```

   with:

   ```html
   <span class="comment-badge mark-chip mark-fixed" data-mark-chip>fixed</span>
   ```

   Keep the rest of the card unchanged. Match on the full `data-mark-id`, not on the snippet or the file path, because several findings can share a file. The viewer hides marked cards while **Hide marked** is enabled, which is its default; uncheck it to see the fixed card and its chip.

   The viewer also replays marks saved in browser `localStorage` after load, and that replay overrides the file. If the report was opened in this browser and the finding is stored as `ignored` or cleared, remove its entry from `ocr-viewer-marks:<page path>` so the file mark is what shows.

10. **Delete the report when done.** Re-run the listing from step 2. If no unmarked finding remains besides `low` ones, delete the report file. Keep it while any other unmarked finding remains, even a skipped one.

11. **Report** the finding id, severity, category, and source path, what changed, the version bump applied or why none was needed, the verification result, that the card now carries `data-mark="fixed"`, and whether the report was deleted.

## Blocking behavior

- No report: report that there is nothing to address and stop.
- No unmarked findings besides `low` ones: delete the report, say so, and stop.
- Code does not match the finding: skip it, continue with the next candidate, and report the mismatch.
- Verification cannot run in this environment: say which command was unavailable and why, and report the finding as addressed but unverified.
