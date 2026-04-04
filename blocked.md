# Blocked: remove-overview requirement

## What I tried
1. Deleted internal/cmd/overview.go - This removed OverviewCmd, Overview, OverviewComponent types, and helper functions (loadOverview, buildOverviewPrompt, buildComponentPrompt)
2. Deleted internal/cmd/overview_test.go, internal/cmd/overview-instructions.md, internal/cmd/component-review-instructions.md
3. Removed Overview field from Cmd struct in internal/cmd/cmd.go

## Why it didn't work
The review.go file depends on types and functions from overview.go:
- `OverviewComponent` type used in shuffleComponents()
- `Overview` type used in runOverview(), runReview(), printDetectedComponents()
- `loadOverview()` function called in runOverview()
- `buildOverviewPrompt()` function called in runOverview()
- `buildComponentPrompt()` function called in runReview()

After deleting overview.go, the code fails to compile with:
- undefined: OverviewComponent
- undefined: Overview  
- undefined: loadOverview
- undefined: buildOverviewPrompt
- undefined: buildComponentPrompt

## Why tests cannot run
The build fails, so Go tests cannot compile or run. The requirement states tests should pass, but this is impossible without first addressing the review.go dependencies.

## How it could be fixed
The next requirement (rewrite-review) would need to be implemented first to remove the dependencies on overview types/functions from review.go. Alternatively, the types and helper functions needed by review.go could be moved to review.go before deleting overview.go, but this would violate the explicit requirement to "Delete internal/cmd/overview.go".