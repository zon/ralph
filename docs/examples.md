# Usage Examples

This guide provides practical examples of using Ralph in different scenarios.

## Table of Contents

- [Basic Usage](#basic-usage)
- [With Services](#with-services)
- [Web Development](#web-development)
- [API Development](#api-development)
- [Database Migrations](#database-migrations)
- [Multi-Step Features](#multi-step-features)
- [Bug Fixes](#bug-fixes)
- [Refactoring](#refactoring)
- [Testing](#testing)

## Basic Usage

### Simple Feature Addition

**Scenario:** Add a new utility function to your project.

**Project file: `projects/add-utility.yaml`**
```yaml
name: add-utility
description: Add string formatting utility function

requirements:
  - description: String Formatting Function
    steps:
      - Create utils/format.go file
      - Implement FormatUserName function
      - Add unit tests
    passing: false
```

**Execute:**
```bash
# Preview first
ralph run projects/add-utility.yaml --dry-run

# Run full workflow
ralph run projects/add-utility.yaml
```

### Quick Bug Fix

**Scenario:** Fix a simple bug without full orchestration.

**Project file: `projects/fix-validation.yaml`**
```yaml
name: fix-validation
description: Fix email validation regex

requirements:
  - description: Email Validation Bug
    steps:
      - Update email regex in validators.go
      - Add test cases for edge cases
      - Verify all tests pass
    passing: false
```

**Execute:**
```bash
# Single iteration only
ralph once projects/fix-validation.yaml

# Check results
git diff
git status

# If satisfied, commit and push manually
git add .
git commit -m "Fix email validation regex"
git push
```

## With Services

### Development with Database

**Scenario:** Develop features that require a running database.

**Configuration: `.ralph/config.yaml`**
```yaml
services:
  - name: postgres
    command: docker
    args: [compose, up, -d, postgres]
    port: 5432
```

**Project file: `projects/add-user-model.yaml`**
```yaml
name: add-user-model
description: Create User database model

requirements:
  - category: model
    description: User Model
    steps:
      - Create User model with Gorm
      - Add fields: username, email, created_at
      - Create migration
    passing: false
    
  - category: testing
    description: Model Tests
    steps:
      - Write tests for User creation
      - Test validation rules
      - Test database constraints
    passing: false
```

**Execute:**
```bash
# Database starts automatically
ralph once projects/add-user-model.yaml

# Database stops when complete
```

### Full Stack Development

**Scenario:** Develop with frontend, backend, and database running.

**Configuration: `.ralph/config.yaml`**
```yaml
services:
  - name: postgres
    command: docker
    args: [compose, up, -d, postgres]
    port: 5432
    
  - name: api
    command: npm
    args: [run, dev:api]
    port: 3000
    
  - name: frontend
    command: npm
    args: [run, dev:frontend]
    port: 8080
```

**Project file: `projects/user-profile-page.yaml`**
```yaml
name: user-profile-page
description: Implement user profile page

requirements:
  - category: backend
    description: Profile API Endpoint
    steps:
      - Create GET /api/users/:id endpoint
      - Return user profile data
      - Add authentication check
    passing: false
    
  - category: frontend
    description: Profile Page UI
    steps:
      - Create ProfilePage component
      - Fetch user data from API
      - Display profile information
      - Add edit button (non-functional)
    passing: false
```

**Execute:**
```bash
ralph once projects/user-profile-page.yaml
```

## Web Development

### React Component

**Project file: `projects/add-search-component.yaml`**
```yaml
name: add-search-component
description: Add search bar component to header

requirements:
  - category: component
    description: SearchBar Component
    steps:
      - Create SearchBar.tsx in components/
      - Add input with search icon
      - Add onChange handler
      - Add debouncing for search input
    passing: false
    
  - category: styling
    description: Component Styling
    steps:
      - Add CSS module for SearchBar
      - Match design system colors
      - Add responsive layout
    passing: false
    
  - category: integration
    description: Add to Header
    steps:
      - Import SearchBar in Header component
      - Position in header layout
      - Connect to search functionality
    passing: false
```

**Execute:**
```bash
ralph run projects/add-search-component.yaml --max-iterations 5
```

### API Endpoint

**Project file: `projects/create-posts-endpoint.yaml`**
```yaml
name: create-posts-endpoint
description: Add REST API for blog posts

requirements:
  - category: model
    description: Post Model
    steps:
      - Create Post model (title, content, author_id, created_at)
      - Add model validation
    passing: false
    
  - category: endpoints
    description: CRUD Endpoints
    steps:
      - GET /api/posts - list all posts
      - GET /api/posts/:id - get single post
      - POST /api/posts - create post (auth required)
      - PUT /api/posts/:id - update post (auth required)
      - DELETE /api/posts/:id - delete post (auth required)
    passing: false
    
  - category: testing
    description: API Tests
    steps:
      - Write integration tests for all endpoints
      - Test authentication requirements
      - Test error cases
    passing: false
```

**Configuration: `.ralph/config.yaml`**
```yaml
services:
  - name: postgres
    command: docker
    args: [compose, up, -d, postgres]
    port: 5432
  - name: api
    command: npm
    args: [run, dev]
    port: 3000
```

**Execute:**
```bash
ralph run projects/create-posts-endpoint.yaml
```

## Database Migrations

### Add New Column

**Project file: `projects/add-email-verification.yaml`**
```yaml
name: add-email-verification
description: Add email verification to users

requirements:
  - category: migration
    description: Database Migration
    steps:
      - Create migration to add email_verified boolean column
      - Create migration to add email_verified_at timestamp
      - Set default email_verified to false
    passing: false
    
  - category: model
    description: Update User Model
    steps:
      - Add EmailVerified field to User struct
      - Add EmailVerifiedAt field
      - Update model tests
    passing: false
    
  - category: logic
    description: Verification Logic
    steps:
      - Add MarkEmailVerified method to User
      - Update registration to set email_verified = false
      - Add verification token generation
    passing: false
```

**Execute:**
```bash
ralph run projects/add-email-verification.yaml
```

## Multi-Step Features

### Authentication System

**Project file: `projects/implement-auth.yaml`**
```yaml
name: implement-auth
description: Implement JWT-based authentication system

requirements:
  - category: model
    description: User Model
    steps:
      - Create User model with password hashing
      - Add SetPassword method with bcrypt
      - Add VerifyPassword method
    passing: false
    
  - category: jwt
    description: JWT Token Generation
    steps:
      - Create JWT helper functions
      - Implement GenerateToken(user)
      - Implement ValidateToken(token)
      - Add refresh token support
    passing: false
    
  - category: endpoints
    description: Auth Endpoints
    steps:
      - POST /api/auth/register - create user
      - POST /api/auth/login - return JWT token
      - POST /api/auth/refresh - refresh access token
      - GET /api/auth/me - get current user (protected)
    passing: false
    
  - category: middleware
    description: Auth Middleware
    steps:
      - Create authentication middleware
      - Extract and validate JWT from headers
      - Attach user to request context
      - Handle invalid/expired tokens
    passing: false
    
  - category: testing
    description: Comprehensive Tests
    steps:
      - Test user registration
      - Test login flow
      - Test protected endpoints
      - Test token expiration
      - Test invalid credentials
    passing: false
```

**Execute:**
```bash
# This is complex, allow more iterations
ralph run projects/implement-auth.yaml --max-iterations 20
```

## Bug Fixes

### Memory Leak Fix

**Project file: `projects/fix-memory-leak.yaml`**
```yaml
name: fix-memory-leak
description: Fix memory leak in event handler cleanup

requirements:
  - category: investigation
    description: Identify Memory Leak
    steps:
      - Profile application memory usage
      - Identify component with leak
      - Find missing cleanup code
    passing: false
    
  - category: fix
    description: Implement Fix
    steps:
      - Add cleanup in component unmount
      - Remove event listeners properly
      - Clear intervals/timeouts
    passing: false
    
  - category: verification
    description: Verify Fix
    steps:
      - Re-run memory profiling
      - Verify memory usage is stable
      - Add test to prevent regression
    passing: false
```

**Execute:**
```bash
ralph once projects/fix-memory-leak.yaml
```

## Refactoring

### Extract Reusable Component

**Project file: `projects/refactor-modal.yaml`**
```yaml
name: refactor-modal
description: Extract modal into reusable component

requirements:
  - category: component
    description: Create Modal Component
    steps:
      - Create Modal.tsx component
      - Add props (isOpen, onClose, title, children)
      - Add styling and animations
    passing: false
    
  - category: refactoring
    description: Replace Existing Modals
    steps:
      - Replace UserEditModal with new Modal
      - Replace DeleteConfirmModal with new Modal
      - Replace SettingsModal with new Modal
      - Remove duplicate modal code
    passing: false
    
  - category: testing
    description: Add Tests
    steps:
      - Test modal open/close behavior
      - Test prop handling
      - Test accessibility features
    passing: false
```

**Execute:**
```bash
ralph run projects/refactor-modal.yaml
```

## Testing

### Add Missing Tests

**Project file: `projects/add-validation-tests.yaml`**
```yaml
name: add-validation-tests
description: Add comprehensive tests for validation functions

requirements:
  - category: unit-tests
    description: Validation Unit Tests
    steps:
      - Test email validation with valid emails
      - Test email validation with invalid emails
      - Test phone number validation
      - Test password strength validation
      - Test credit card validation
    passing: false
    
  - category: edge-cases
    description: Edge Case Tests
    steps:
      - Test empty string inputs
      - Test null/undefined inputs
      - Test very long inputs
      - Test special characters
      - Test unicode characters
    passing: false
    
  - category: coverage
    description: Coverage Report
    steps:
      - Run tests with coverage
      - Ensure 90%+ coverage for validators
      - Document any uncovered edge cases
    passing: false
```

**Execute:**
```bash
ralph run projects/add-validation-tests.yaml
```

## Advanced Workflows

### Iterative Development

**Scenario:** Work on a feature manually between Ralph iterations.

```bash
# Start first iteration
ralph once projects/feature.yaml

# Review changes
git diff

# Make manual adjustments
vim src/component.tsx

# Run another iteration
ralph once projects/feature.yaml

# Repeat until satisfied
```

### Multiple Features in Parallel

**Scenario:** Work on multiple independent features.

```bash
# Terminal 1: Feature A
ralph run projects/feature-a.yaml

# Terminal 2: Feature B (different branch)
ralph run projects/feature-b.yaml

# Both create separate branches and PRs
```

### Dry-Run for Planning

**Scenario:** Plan complex changes before executing.

```bash
# Preview what would happen
ralph run projects/complex-feature.yaml --dry-run --verbose

# Review the plan
# Adjust project file if needed

# Execute when ready
ralph run projects/complex-feature.yaml
```

### Custom Development Instructions

**Scenario:** Provide project-specific guidance to AI.

**Create: `docs/develop-instructions.md`**
```markdown
# Development Instructions

## Code Style
- Use TypeScript strict mode
- Prefer functional components
- Use custom hooks for logic
- Follow Airbnb style guide

## Testing
- Write tests for all components
- Use React Testing Library
- Minimum 80% coverage

## Naming
- Components: PascalCase
- Files: kebab-case
- Functions: camelCase
```

**Execute:**
```bash
# Ralph automatically includes develop-instructions.md in prompts
ralph once projects/feature.yaml
```

## Tips and Tricks

### 1. Start Small

Begin with simple, focused requirements:
```yaml
requirements:
  - description: Just one thing
    passing: false
```

### 2. Use Categories

Organize complex features:
```yaml
requirements:
  - category: backend
    description: API changes
    passing: false
  - category: frontend
    description: UI changes
    passing: false
  - category: testing
    description: Test coverage
    passing: false
```

### 3. Specific Steps

Provide clear implementation steps:
```yaml
requirements:
  - description: Feature X
    steps:
      - Specific step 1
      - Specific step 2
      - Specific step 3
    passing: false
```

### 4. Dry-Run First

Always preview complex operations:
```bash
ralph run project.yaml --dry-run
```

### 5. Verbose Mode for Debugging

Get detailed logs:
```bash
ralph run project.yaml --verbose
```

### 6. Skip Services When Needed

If services are slow or unnecessary:
```bash
ralph once project.yaml --no-services
```

### 7. Adjust Iterations

Match iterations to complexity:
```bash
# Simple fix
ralph run project.yaml --max-iterations 3

# Complex feature
ralph run project.yaml --max-iterations 20
```

## More Examples

See the `examples/` directory for more sample project files:
- `examples/sample-project.yaml` - Comprehensive example
- `examples/config.example.yaml` - Service configuration
- `examples/secrets.example.yaml` - API key setup

## Need Help?

- [README](../README.md) - Full documentation
- [Configuration Guide](configuration.md) - Config options
- [Quick Start](quick-start.md) - Get started quickly
- [GitHub Issues](https://github.com/zon/ralph/issues) - Report problems
