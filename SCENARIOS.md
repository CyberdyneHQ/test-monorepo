# Test Monorepo — SCA Target Detection Scenarios

This repo is designed to test every way a monorepo can arrange lockfiles, manifests,
and workspace configs. It tests asgard's `RepositoryTarget.detect()` for each subrepo.

## Root-level files (shared across subrepos)

| File | Purpose |
|------|---------|
| `package.json` | npm workspace config: `["apps/web", "apps/mobile", "apps/dashboard", "packages/shared-utils"]` |
| `package-lock.json` | Root npm lockfile |
| `yarn.lock` | Root yarn lockfile (COMPETING with npm!) |
| `pnpm-workspace.yaml` | pnpm workspace config: `["apps/web", "apps/mobile", "packages/shared-utils"]` |
| `Cargo.toml` | Rust workspace: `members = ["crates/core", "crates/utils"]` |
| `Cargo.lock` | Root cargo lockfile |
| `go.work` | Go workspace: `use ./services/api, ./services/gateway` |
| `pyproject.toml` | uv workspace: `members = ["backend/api"]` |
| `uv.lock` | Root uv lockfile |

**NOTE:** Asgard does NOT read workspace configs. Detection is BLIND — it pairs root lockfile
+ subrepo manifest based solely on file existence + PM eligibility. Workspace configs exist
here to document what SHOULD be validated but currently isn't.

---

## Scenario Matrix

### 1. `apps/web` — Workspace member, no own lockfile
- **Files:** `package.json`
- **Workspace listed in:** root `package.json`, `pnpm-workspace.yaml`
- **Expected targets (asgard):**
  - `(package-lock.json, apps/web/package.json, npm)` — root lockfile pairing (BLIND)
  - `(yarn.lock, apps/web/package.json, yarn)` — ALSO paired (BLIND, competing PM!)
- **Issue:** Asgard creates TWO targets (npm + yarn) because both root lockfiles exist.
  It doesn't know which PM actually manages this subrepo.

### 2. `apps/web-admin` — Has OWN lockfile + partial prefix overlap with "web"
- **Files:** `package-lock.json`, `package.json`
- **Workspace listed in:** NONE (self-contained, not in root workspaces)
- **Expected targets (asgard):**
  - `(apps/web-admin/package-lock.json, apps/web-admin/package.json, npm)` — local target
- **Tests:** Self-contained subrepo with own lockfile. Local lockfile means no root pairing.
  Must NOT leak to `apps/web/` (prefix overlap: "web" vs "web-admin").

### 3. `apps/mobile` — Workspace member, no own lockfile
- **Files:** `package.json`
- **Workspace listed in:** root `package.json`, `pnpm-workspace.yaml`
- **Expected targets (asgard):**
  - `(package-lock.json, apps/mobile/package.json, npm)` — root lockfile pairing
  - `(yarn.lock, apps/mobile/package.json, yarn)` — competing root lockfile

### 4. `apps/dashboard` — Nested workspace (workspace within workspace)
- **Files:** `package.json` (has its own `"workspaces": ["packages/*"]`), `yarn.lock`
- **Sub-project:** `packages/ui/package.json`
- **Expected targets (asgard):**
  - `(apps/dashboard/yarn.lock, apps/dashboard/package.json, yarn)` — local target
- **Tests:** Asgard won't detect `apps/dashboard/packages/ui/package.json` as a separate
  target since there's no lockfile next to it. The nested workspace config is invisible
  to asgard. Root lockfile should be deduplicated since local yarn.lock claims the manifest.

### 5. `packages/shared-utils` — Workspace member, no own lockfile
- **Files:** `package.json`
- **Workspace listed in:** root `package.json`, `pnpm-workspace.yaml`
- **Expected targets (asgard):**
  - `(package-lock.json, packages/shared-utils/package.json, npm)` — root lockfile
  - `(yarn.lock, packages/shared-utils/package.json, yarn)` — competing root

### 6. `packages/config` — NOT in any workspace config
- **Files:** `package.json`
- **Workspace listed in:** NONE (not in root package.json workspaces, not in pnpm-workspace.yaml)
- **Expected targets (asgard):**
  - `(package-lock.json, packages/config/package.json, npm)` — FALSE POSITIVE!
  - `(yarn.lock, packages/config/package.json, yarn)` — FALSE POSITIVE!
- **Issue:** Asgard blindly pairs because detection is not workspace-aware.
  The root lockfile does NOT actually cover this package.

### 7. `services/api` — Go with own lockfile
- **Files:** `go.mod`, `go.sum`
- **Workspace listed in:** `go.work`
- **Expected targets (asgard):**
  - `(services/api/go.sum, services/api/go.mod, go_mod)` — local target
- **Tests:** Go is NOT in ROOT_LOCKFILE_ELIGIBLE_PACKAGE_MANAGERS, so no root pairing.

### 8. `services/gateway` — Go WITHOUT lockfile
- **Files:** `go.mod` only (no `go.sum`)
- **Workspace listed in:** `go.work`
- **Expected targets (asgard):**
  - NONE — no lockfile means no target detected
- **Issue:** A valid Go module with no go.sum won't be detected. go.work is invisible.

### 9. `services/auth` — Mixed ecosystem (Python + JS)
- **Files:** `requirements.txt`, `pyproject.toml`, `package.json`, `package-lock.json`
- **Expected targets (asgard):**
  - `(services/auth/requirements.txt, services/auth/pyproject.toml, requirements_txt)` — Python
  - `(services/auth/package-lock.json, services/auth/package.json, npm)` — JS
- **Tests:** Two ecosystems in one subrepo, each gets its own target.

### 10. `crates/core` — Rust workspace member
- **Files:** `Cargo.toml`
- **Workspace listed in:** root `Cargo.toml`
- **Expected targets (asgard):**
  - `(Cargo.lock, crates/core/Cargo.toml, cargo)` — root lockfile pairing (Cargo is eligible)
- **Tests:** Cargo workspace pattern correctly paired via blind detection.

### 11. `crates/utils` — Rust workspace member
- **Files:** `Cargo.toml`
- **Workspace listed in:** root `Cargo.toml`
- **Expected targets (asgard):**
  - `(Cargo.lock, crates/utils/Cargo.toml, cargo)` — root lockfile pairing

### 12. `backend/api` — Python uv workspace member
- **Files:** `pyproject.toml`
- **Workspace listed in:** root `pyproject.toml` (uv workspace)
- **Expected targets (asgard):**
  - `(uv.lock, backend/api/pyproject.toml, uv)` — root lockfile pairing (uv is eligible)
- **Tests:** uv workspace pattern correctly paired via blind detection.

### 13. `backend/workers` — Python poetry (own lockfile, ineligible PM)
- **Files:** `poetry.lock`, `pyproject.toml`
- **Expected targets (asgard):**
  - `(backend/workers/poetry.lock, backend/workers/pyproject.toml, poetry)` — local target
- **Tests:** Poetry is NOT in ROOT_LOCKFILE_ELIGIBLE_PACKAGE_MANAGERS.
  Has own lockfile, so root uv.lock should not also pair.

### 14. `infra` — No package manager files at all
- **Files:** `main.tf` (Terraform)
- **Expected targets (asgard):** NONE
- **Tests:** Subrepos without any PM files produce no targets.

### 15. `dotnet/WebApp` — .NET with packages.lock.json + csproj
- **Files:** `WebApp.csproj`, `packages.lock.json`
- **Expected targets (asgard):**
  - `(dotnet/WebApp/packages.lock.json, <none>, nuget)` — packages.lock.json has empty manifest mapping
- **Note:** The csproj in same dir as packages.lock.json gets skipped by _detect_csproj_targets.

### 16. `dotnet/Library` — .NET with csproj only (no lockfile)
- **Files:** `Library.csproj`
- **Expected targets (asgard):**
  - `(dotnet/Library/Library.csproj, dotnet/Library/Library.csproj, nuget)` — csproj-as-target fallback
- **Tests:** _detect_csproj_targets picks up csproj files without packages.lock.json.

### 17. `legacy/php-app` — PHP Composer
- **Files:** `composer.lock`, `composer.json`
- **Expected targets (asgard):**
  - `(legacy/php-app/composer.lock, legacy/php-app/composer.json, packagist)` — local target
- **Tests:** Composer is NOT in ROOT_LOCKFILE_ELIGIBLE_PACKAGE_MANAGERS. Standard local detection.

### 18. `legacy/ruby-app` — Ruby Bundler
- **Files:** `Gemfile.lock`, `Gemfile`
- **Expected targets (asgard):**
  - `(legacy/ruby-app/Gemfile.lock, legacy/ruby-app/Gemfile, ruby_gems)` — local target
- **Tests:** Ruby is NOT in ROOT_LOCKFILE_ELIGIBLE_PACKAGE_MANAGERS. Standard local detection.

### 19. `java/spring-app` — Maven
- **Files:** `pom.xml`
- **Expected targets (asgard):**
  - `(java/spring-app/pom.xml, java/spring-app/pom.xml, maven)` — pom.xml is both lockfile and manifest
- **Tests:** Maven's pom.xml maps to itself in LOCKFILE_TO_MANIFEST_MAP.

### 20. `java/gradle-app` — Gradle with lockfile
- **Files:** `gradle.lockfile`, `build.gradle`
- **Expected targets (asgard):**
  - `(java/gradle-app/gradle.lockfile, java/gradle-app/build.gradle, gradle)` — local target

### 21. `orphaned/stale-project` — Orphaned lockfile (manifest deleted)
- **Files:** `yarn.lock` only (no package.json)
- **Expected targets (asgard):**
  - `(orphaned/stale-project/yarn.lock, <none>, yarn)` — lockfile with no manifest
- **Tests:** Asgard creates a target even without a manifest (manifest_path=None).

### 22. `admin` — Path prefix overlap test (admin vs admin-panel)
- **Files:** `yarn.lock`, `package.json`
- **Expected targets (asgard):**
  - `(admin/yarn.lock, admin/package.json, yarn)` — local target
- **Tests:** Must NOT include files from `admin-panel/`. The trailing `/` in
  `subrepo_prefix = repository.subrepo.path + "/"` prevents this.

### 23. `admin-panel` — Path prefix overlap test
- **Files:** `package-lock.json`, `package.json`
- **Expected targets (asgard):**
  - `(admin-panel/package-lock.json, admin-panel/package.json, npm)` — local target
- **Tests:** Must NOT include files from `admin/`.

### 24. `clients/internal/portal` — Deeply nested subrepo path
- **Files:** `pnpm-lock.yaml`, `package.json`
- **Workspace listed in:** NONE (self-contained, has own pnpm lockfile)
- **Expected targets (asgard):**
  - `(clients/internal/portal/pnpm-lock.yaml, clients/internal/portal/package.json, pnpm)` — local
- **Tests:** Multi-level deep subrepo path works correctly with prefix filtering.

### 25. `tools/bundler` — Bun ecosystem
- **Files:** `bun.lock`, `package.json`
- **Expected targets (asgard):**
  - `(tools/bundler/bun.lock, tools/bundler/package.json, bun)` — local target
- **Tests:** Bun lockfile detection. Bun IS in ROOT_LOCKFILE_ELIGIBLE_PACKAGE_MANAGERS
  but local lockfile means root pairing gets deduplicated.

### 26. `scripts/etl` — Pipfile ecosystem
- **Files:** `Pipfile.lock`, `Pipfile`
- **Expected targets (asgard):**
  - `(scripts/etl/Pipfile.lock, scripts/etl/Pipfile, pipfile)` — local target
- **Tests:** Pipfile is NOT in ROOT_LOCKFILE_ELIGIBLE_PACKAGE_MANAGERS. Standard detection.

### 27. `scripts/ml-pipeline` — PDM ecosystem
- **Files:** `pdm.lock`, `pyproject.toml`
- **Expected targets (asgard):**
  - `(scripts/ml-pipeline/pdm.lock, scripts/ml-pipeline/pyproject.toml, pdm)` — local target
- **Tests:** PDM is NOT in ROOT_LOCKFILE_ELIGIBLE_PACKAGE_MANAGERS. Standard detection.

### 28. `experiments/dual-lock` — Competing lockfiles, same ecosystem
- **Files:** `package-lock.json`, `yarn.lock`, `package.json`
- **Expected targets (asgard):**
  - `(experiments/dual-lock/package-lock.json, experiments/dual-lock/package.json, npm)` — npm target
  - `(experiments/dual-lock/yarn.lock, experiments/dual-lock/package.json, yarn)` — yarn target
- **Issue:** Creates TWO targets for the same manifest with different PMs.
  Only one PM actually manages this project, but asgard can't tell which.

---

## Known Blind Detection Issues

These are problems caused by asgard NOT reading workspace configs:

1. **False positive pairing (Scenario 6):** `packages/config` is NOT in any workspace,
   but asgard pairs it with root lockfiles anyway.

2. **Competing PM ambiguity (Scenarios 1, 3, 5):** When root has both `package-lock.json`
   AND `yarn.lock`, workspace members get TWO targets (one npm, one yarn).
   Only one is real.

3. **Nested workspace blindness (Scenario 4):** `apps/dashboard/packages/ui/package.json`
   is a workspace member of dashboard's inner workspace, but asgard can't detect it as
   a target because there's no lockfile next to it and it doesn't understand nested workspaces.

4. **go.work ignorance (Scenario 8):** `services/gateway` is a valid Go module listed
   in `go.work` but has no `go.sum` — asgard sees nothing.

5. **Root targets for the monorepo itself:** The root `package-lock.json` and `yarn.lock`
   would also create targets for the root repo itself. If the monorepo root is treated
   as a repository (not a subrepo), it gets targets for ALL root lockfiles.
