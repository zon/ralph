# Comment Instructions

You have been triggered by a GitHub pull request comment.

Pull request details:
- Repository: {{.RepoOwner}}/{{.RepoName}}
- PR number: #{{.PRNumber}}
- Branch: {{.PRBranch}}

The comment text is:

---
{{.CommentBody}}
---

Your task:
1. Read the comment carefully.
2. If the comment asks a question, answer it by posting a GitHub PR comment.
3. If the comment requests code changes, implement them, then commit and push the changes.
4. After completing your work, post a GitHub PR comment summarising what you did.

When posting PR comments use the gh CLI with a heredoc to avoid shell interpretation of special characters:
  gh pr comment {{.PRNumber}} --body "$(cat <<'EOF'
<your summary here>
EOF
)"
