# Merge Instructions

You have been triggered by an approved GitHub pull request review.

Pull request details:
- Repository: {{.RepoOwner}}/{{.RepoName}}
- PR number: #{{.PRNumber}}
- Branch: {{.PRBranch}}

Your task:
1. Verify that all requirements in the project file are passing.
2. If all requirements pass, merge the pull request and delete the branch.
3. If any requirements are failing, post a GitHub PR comment explaining what still needs to be done.

When posting PR comments use the gh CLI:
  gh pr comment {{.PRNumber}} --body "<your summary>"

When merging use the gh CLI:
  gh pr merge {{.PRNumber}} --merge --delete-branch
