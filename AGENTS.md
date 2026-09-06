## Agent skills

### Issue tracker

Issues live on GitHub (uses `gh` CLI). See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context repo layout (`CONTEXT.md` + `docs/adr/` at root). See `docs/agents/domain.md`.

### Git & PR Workflow

- **No Direct Pushes to `main`**: Pushing code directly to the `main` branch is strictly prohibited.
- **PR Required**: All code changes, bug fixes, and features must be developed on feature/topic branches and merged into `main` strictly via Pull Requests (PRs).


<!-- START AGENT-STANDARD: CLEAN-CODE -->
## Code Quality, Maintainability & Documentation
- Enforce all 5 SOLID principles: Single Responsibility (SRP) per module, Open/Closed (OCP) via extension points, Liskov Substitution (LSP) for behavioral subtyping, Interface Segregation (ISP) with small role-specific contracts, and Dependency Inversion (DIP) via injected abstractions.
- Apply pragmatic DRY (Don't Repeat Yourself) to consolidate business rules into single sources of truth, while adhering to YAGNI (You Aren't Gonna Need It) and the AHA principle (Avoid Hasty Abstractions). Do NOT write speculative abstractions, dead code, or unused generic parameters.
- Use guard clauses (early exit returns/throws) at the top of functions instead of deeply nested `if-else` blocks to maintain low cyclomatic complexity ($\le 3$ nesting levels).
- Enforce strict file length boundaries: target 200300 lines of code (LOC) per file (Robert C. Martin "Newspaper Metaphor"), enforce a soft cap at 400 LOC (SmartBear/Cisco study: review defect detection drops sharply past 400 LOC), and treat 500 LOC as an absolute hard ceiling. Exclude auto-generated code (lockfiles, OpenAPI/Protobuf artifacts) and large test fixtures.
- Write self-documenting code with domain-aligned naming. Inline comments MUST explain non-obvious business rationale (*why*), never restating *what* readable code already expresses.
- Keep inline docstrings, API contracts, and external specifications (OpenAPI 3.1, Protocol Buffers, GraphQL) 100% synchronized whenever signatures or data models change.
<!-- END AGENT-STANDARD: CLEAN-CODE -->

