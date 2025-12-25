# Development Workflow and Git Practices

## Commit Strategy

### Small, Frequent Commits
- **Commit small, commit often**
- Each commit should represent a single logical change
- Commits must be well-tested before committing
- Avoid large commits or massive file changes per PR

### Commit Guidelines
- **Follow Conventional Commits specification** ([conventionalcommits.org](https://www.conventionalcommits.org/en/v1.0.0/))
- Write clear, descriptive commit messages with proper type prefixes
- Ensure all tests pass before committing
- Include relevant test coverage with each feature commit

#### Conventional Commit Format
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### Commit Types
- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code (white-space, formatting, etc)
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **build**: Changes that affect the build system or external dependencies
- **ci**: Changes to CI configuration files and scripts
- **chore**: Other changes that don't modify src or test files
- **revert**: Reverts a previous commit

#### Breaking Changes
- Use `!` after the type/scope to indicate breaking changes
- Include `BREAKING CHANGE:` in the footer with description

```bash
# Good commit examples
git commit -m "feat(fleet): add Fleet custom resource definition"
git commit -m "test(controller): add unit tests for fleet controller reconcile logic"
git commit -m "fix(status): handle nil pointer in fleet status update"
git commit -m "feat(api)!: change Fleet spec structure

BREAKING CHANGE: Fleet.spec.replicas is now Fleet.spec.size"
git commit -m "docs(readme): add installation instructions"
git commit -m "refactor(builder): simplify FleetBuilder interface"
git commit -m "ci(github): add automated testing workflow"
```

## Git Workflow Strategy

### Trunk-Based Development
- **Use trunk-based development** with main as the primary branch
- **Commit often and PR often** to maintain small, reviewable changes
- **Merge directly into main** after code review and CI validation
- **Avoid long-lived feature branches** that diverge significantly from main
- **Use short-lived branches** for individual features or fixes (typically 1-3 days)

### Branch Naming Convention
- **Feature branches**: `feature/short-description` or `feat/issue-123`
- **Bug fixes**: `fix/short-description` or `bugfix/issue-456`
- **Chores**: `chore/short-description` (dependencies, tooling, etc.)
- **Documentation**: `docs/short-description`

### Branch and Push Strategy

#### Confirmation Requirements
- **Always prompt before committing**
- **Always prompt separately before pushing branches**
- Confirm approach before making large refactors
- Discuss technical debt before addressing it

#### Workflow Steps
1. Create short-lived branch from main: `git checkout -b feature/vm-lifecycle`
2. Make changes and write tests
3. Run tests locally to ensure they pass
4. **PROMPT**: "Ready to commit these changes?"
5. After commit confirmation, **PROMPT**: "Ready to push this branch?"
6. Create PR immediately after pushing (even if work in progress)
7. Merge to main after review and CI validation
8. Delete feature branch after merge

### Pull Request Strategy
- **Create PRs early and often** - even for work in progress
- **Use draft PRs** for ongoing work to get early feedback
- **Keep PRs small and focused** - ideally under 400 lines of changes
- **One logical change per PR** - easier to review and revert if needed
- **Merge frequently** to avoid conflicts and integration issues

### Continuous Integration
- **All PRs must pass CI** before merging to main
- **Run full test suite** on every PR
- **Automated checks**: linting, formatting, security scanning
- **No direct pushes to main** - all changes via PR
- **Require at least one approval** for PR merges

### Trunk-Based Development Benefits
- **Faster integration**: Reduces merge conflicts and integration issues
- **Continuous feedback**: Early detection of problems through frequent integration
- **Simplified workflow**: No complex branching strategies to manage
- **Better collaboration**: Everyone works from the same recent codebase
- **Easier releases**: Main branch is always in a releasable state

### Feature Flags for Large Changes
- **Use feature flags** for large features that can't be completed in small PRs
- **Toggle incomplete features** to keep main branch stable
- **Gradual rollout** of new functionality
- **Quick rollback** capability without code changes

```go
// Example: Feature flag usage
if featureFlags.IsEnabled("new-vm-lifecycle") {
    return r.newVMLifecycleHandler(ctx, vm)
}
return r.legacyVMLifecycleHandler(ctx, vm)
```

### Main Branch Protection
- **Main branch is always deployable** - never broken
- **Fast-forward merges preferred** when possible
- **Squash commits** for cleaner history when appropriate
- **Immediate rollback capability** if issues are detected

## Technical Debt Management
### Issue Identification
- When technical debt is discovered, pause development
- Create issues for deferred technical debt
- Don't accumulate technical debt without tracking

### Decision Process
1. Identify technical debt or improvement opportunity
2. **PROMPT**: "Found technical debt: [description]. Should we create an issue or address it now?"
3. Wait for guidance on prioritization
4. Document decision and reasoning

## Bug and Refactor Handling

### Large Refactor Protocol
- If a bug requires significant refactoring, stop and assess
- **PROMPT**: "This bug requires a larger refactor: [description]. Confirm approach?"
- Get explicit approval before proceeding with major changes
- Break large refactors into smaller, reviewable chunks

### Bug Triage Process
1. Identify and document the bug
2. Assess scope of required changes
3. If changes are substantial, prompt for confirmation
4. Implement fix with appropriate test coverage
5. Verify fix doesn't introduce regressions

## Code Review Readiness

### PR Preparation
- Ensure all tests pass locally
- Run linting and formatting tools
- Include test coverage for new functionality
- Keep PRs focused and reviewable
- Write clear PR descriptions explaining changes

### Quality Gates
- All code must be tested
- No failing tests in commits
- Follow Go best practices and project conventions
- Include documentation updates when needed