# PRD: Auth & Tenancy Completion

## Overview

The auth and tenancy system for Nori is approximately 70% implemented. The backend models, repositories, services, and middleware guards all exist and are well-tested. However, the middleware isn't wired to routes, admin handlers aren't registered, and the frontend/CLI haven't been built. This PRD covers the remaining work: wiring backend middleware to routes, completing the frontend in Svelte 5 + Tailwind v4, updating the frontend auth infrastructure to match the new backend, and implementing CLI authentication.

## Goals

- Wire all existing auth/authorization middleware to the correct route groups so the backend enforces access control
- Register admin handler routes so user, API key, and space member management is accessible
- Fix security gaps: unauthenticated media/TUS routes, unrestricted space creation, unfiltered space listing
- Build frontend auth flows (login, password change, onboarding, space selector, admin pages) in Svelte 5 + Tailwind v4
- Update the frontend auth store and API client to match the new backend endpoints
- Implement CLI authentication (`nori login`, token storage, API key support)
- Clean up dead code (old permissions.go bitmask system, unused TopNav component, register page)

## Quality Gates

These commands must pass for every user story:

**Backend stories:**
- `go test ./...` - All Go tests pass
- `go vet ./...` - Static analysis passes

**Frontend stories:**
- `npm run build` - SvelteKit build succeeds
- `npm run check` - Svelte check passes (type checking)

## User Stories

### US-001: Wire middleware and register admin routes in app.go

**Description:** As a developer, I want all middleware guards wired to the correct route groups and admin handlers registered so that the backend enforces authentication, authorization, and space scoping.

**Acceptance Criteria:**
- [ ] `RequirePasswordChanged()` middleware is applied to all authenticated route groups except `/auth/change-password` and `/auth/logout`
- [ ] `RequireSpaceAccess()` middleware is applied to space-scoped route groups (SOPs, future station/ticket routes)
- [ ] `authMiddleware` is applied to media upload routes (`/sops/:id/steps/:stepId/media/*`, `/media/*`) and TUS routes (`/api/tus/*`)
- [ ] `AdminUserHandler`, `AdminAPIKeyHandler`, and `AdminSpaceMemberHandler` are instantiated and their `Register*Routes()` methods are called in `app.go`
- [ ] Admin routes are grouped under an authenticated route group with `RequireAdmin()` applied
- [ ] Existing tests continue to pass; new integration tests verify that unauthenticated requests to media/TUS routes return 401

### US-002: Restrict space creation and filter space listing

**Description:** As an admin, I want only admins to create spaces and users to only see spaces they're members of, so that space access is properly controlled.

**Acceptance Criteria:**
- [ ] `POST /api/spaces` returns 403 for non-admin users
- [ ] `GET /api/spaces` returns all account spaces for admins
- [ ] `GET /api/spaces` returns only spaces where the user has a SpaceMember record for non-admin users
- [ ] Tests verify both admin and user behavior for space creation and listing
- [ ] Existing space endpoints (`GET /api/spaces/:id`, `PUT`, `DELETE`) check space access (admin or SpaceMember)

### US-003: Add space scoping to SOP queries

**Description:** As a user, I want SOP queries filtered by active space so that I only see SOPs belonging to my current space.

**Acceptance Criteria:**
- [ ] SOP list and detail queries filter by the `ActiveSpaceID` from the auth context
- [ ] Creating an SOP associates it with the active space
- [ ] Requests without an active space return an appropriate error (e.g., 400 "no active space")
- [ ] Tests verify no cross-space data leakage (user in Space A cannot see SOPs from Space B)
- [ ] If the SOP model doesn't have a `SpaceID` field, add it with a migration

### US-004: Clean up dead backend code

**Description:** As a developer, I want dead code removed so the codebase is clean and doesn't confuse future contributors.

**Acceptance Criteria:**
- [ ] `server/internal/auth/permissions.go` (bitmask permission system) is deleted
- [ ] Any references to the old `auth.Role` type from `permissions.go` are removed
- [ ] The duplicate `GET /user/me` route is removed (replaced by `GET /auth/me`)
- [ ] All tests pass after cleanup

### US-005: Update frontend auth store and API client

**Description:** As a frontend developer, I want the auth store and API client updated to match the new backend so that login, session management, and API calls work correctly.

**Acceptance Criteria:**
- [ ] Auth store `initialize()` calls `/auth/me` instead of `/user/me` and uses the `apiClient` instead of hardcoded `fetch`
- [ ] Auth store handles `mustChangePassword` flag from the login/me response
- [ ] `authApi` removes the `register()` method and updates `getCurrentUser()` to call `/auth/me`
- [ ] API client sends `X-Space-ID` header when an active space is set
- [ ] API client intercepts 401 responses and redirects to `/login`
- [ ] API client intercepts 403 with `MUST_CHANGE_PASSWORD` code and redirects to `/change-password`
- [ ] `User` type is updated to match backend DTO (role: `admin` | `user`, includes `mustChangePassword`)
- [ ] Token is read from HTTP-only cookie (backend sets it) rather than only localStorage

### US-006: Add SvelteKit route guards and server hooks

**Description:** As a user, I want to be redirected to login when unauthenticated and to password change when required, from any page.

**Acceptance Criteria:**
- [ ] `hooks.server.ts` (or `hooks.client.ts`) is created with auth logic that protects all routes except `/login`
- [ ] Unauthenticated users are redirected to `/login` from any page (not just `/`)
- [ ] Users with `mustChangePassword` are redirected to `/change-password` from any page except `/change-password` and `/login`
- [ ] `app.d.ts` types are uncommented and populated with proper `Locals` and `PageData` types
- [ ] The register page (`/register`) is removed since self-registration is disabled
- [ ] Login and register pages are migrated from Svelte 4 event syntax (`on:submit|preventDefault`) to Svelte 5 syntax (`onsubmit`)

### US-007: Build password change page

**Description:** As a user who must change their password, I want a password change form so that I can set a new password and access the application.

**Acceptance Criteria:**
- [ ] New route at `/change-password` with old password, new password, and confirm password fields
- [ ] Client-side validation: all fields required, new passwords must match, minimum length
- [ ] Calls `POST /auth/change-password` endpoint
- [ ] On success, updates auth store with new token/user and redirects to `/` (or onboarding if no spaces)
- [ ] Shows error messages for incorrect old password or server errors
- [ ] Built with Svelte 5 runes syntax and Tailwind v4
- [ ] Page is accessible without full app chrome (sidebar/header) since user hasn't completed auth flow

### US-008: Build onboarding flow for first space creation

**Description:** As an admin logging in for the first time, I want to be guided to create my first space so that I can start using Nori.

**Acceptance Criteria:**
- [ ] After login (and password change if required), if the user is an admin and no spaces exist, show the onboarding flow
- [ ] Onboarding page at `/onboarding` (or modal) with space name input and template selector
- [ ] Calls `POST /api/spaces` to create the space
- [ ] On success, sets the new space as active and redirects to the space view
- [ ] Non-admin users who have no spaces see a message like "No spaces available. Contact your admin."
- [ ] Built with Svelte 5 runes syntax and Tailwind v4

### US-009: Build space selector in header/sidebar

**Description:** As a user with access to multiple spaces, I want to switch between spaces so that I can work in different areas.

**Acceptance Criteria:**
- [ ] Space selector dropdown in the sidebar or header showing accessible spaces
- [ ] Admins see all account spaces; regular users see only their SpaceMember spaces
- [ ] Selecting a space updates the active space in the auth/space store
- [ ] Subsequent API requests include the `X-Space-ID` header with the selected space
- [ ] Selecting a space calls `POST /api/spaces/:id/visit` to update recent spaces
- [ ] Current active space is visually indicated
- [ ] Built with Svelte 5 runes syntax and Tailwind v4

### US-010: Build admin settings pages

**Description:** As an admin, I want settings pages for managing users, API keys, and space members so that I can administer the Nori instance from the UI.

**Acceptance Criteria:**
- [ ] Admin section accessible from sidebar navigation, only visible to admin role
- [ ] **User management page**: list users, create user (email, name, temp password), edit role, deactivate
- [ ] **API key management page**: list keys (name, last used, status), create key (show raw key once), revoke key
- [ ] **Space member management**: within each space's settings, add/remove users
- [ ] All pages call the `/admin/*` endpoints
- [ ] Non-admin users cannot navigate to admin pages (redirect or 404)
- [ ] Built with Svelte 5 runes syntax and Tailwind v4

### US-011: Clean up dead frontend code

**Description:** As a developer, I want unused frontend code removed so the codebase is clean.

**Acceptance Criteria:**
- [ ] `TopNav.svelte` is deleted (its functionality is inlined in the root layout)
- [ ] Register page (`/register`) is deleted (self-registration is disabled)
- [ ] `authApi.register()` method is removed
- [ ] Any imports or references to removed files are cleaned up
- [ ] SvelteKit adapter is switched from `adapter-auto` to `adapter-node` in `svelte.config.js` (since `adapter-node` is already a dependency)

### US-012: Implement `nori login` CLI command

**Description:** As a CLI user, I want to authenticate with `nori login` so that I can use Nori from the command line.

**Acceptance Criteria:**
- [ ] `nori login` command added using cobra
- [ ] Prompts for server URL, email, and password
- [ ] Calls `POST /auth/login` and stores the JWT in `~/.config/nori/credentials`
- [ ] If `mustChangePassword` is true, prompts for new password inline and calls `POST /auth/change-password`
- [ ] Stored credentials are used by subsequent CLI commands via `Authorization: Bearer` header
- [ ] `nori login` can be re-run to refresh credentials
- [ ] Credentials file has restrictive permissions (0600)

### US-013: Add API key support and 401 re-auth to CLI

**Description:** As a CLI user, I want to configure an API key and have the CLI handle expired tokens so that headless and long-running usage works smoothly.

**Acceptance Criteria:**
- [ ] `nori config set api-key nori_...` stores the API key in `~/.config/nori/credentials`
- [ ] CLI prefers API key over JWT if both are configured
- [ ] On 401 response, CLI prompts user to re-authenticate (or exits with clear message if non-interactive)
- [ ] `nori config show` displays current auth method (JWT or API key) without revealing secrets
- [ ] Tests verify credential storage and retrieval

## Functional Requirements

- FR-1: All authenticated routes must validate JWT or API key before processing
- FR-2: All routes except `/auth/login` must enforce `RequirePasswordChanged` when flag is set
- FR-3: All space-scoped routes must enforce `RequireSpaceAccess`
- FR-4: Media upload routes must require authentication
- FR-5: Space creation must be restricted to admin role
- FR-6: Space listing must filter by membership for non-admin users
- FR-7: SOP queries must filter by active space ID
- FR-8: Frontend must redirect unauthenticated users to login from any page
- FR-9: Frontend must redirect users with `mustChangePassword` to password change page
- FR-10: Frontend API client must send `X-Space-ID` header with active space
- FR-11: CLI must store credentials securely in `~/.config/nori/credentials`
- FR-12: Admin pages must only be visible and accessible to admin role users

## Non-Goals (Out of Scope)

- OAuth/SSO authentication (future consideration)
- Per-space roles beyond admin/user (v2)
- Email-based invite flow
- API key scoping per-space (currently account-level)
- Refresh token mechanism (v1 uses 30-day JWT expiry)
- Multi-tenant account support (infrastructure exists but not exercised)
- Frontend unit testing framework setup (build + check is sufficient for now)

## Technical Considerations

- The frontend is already on Svelte 5 + Tailwind v4, so new pages should use runes syntax (`$state`, `$derived`, `$effect`, `$props`) and Tailwind v4 conventions
- The backend middleware guards (`RequireAdmin`, `RequirePasswordChanged`, `RequireSpaceAccess`) exist and are tested; the work is primarily wiring them in `app.go`
- Admin handler code exists and has tests; it just needs to be registered in the route setup
- The CLI doesn't exist yet; a cobra command structure needs to be created from scratch
- The SOP model may need a `SpaceID` foreign key added if it doesn't already have one
- The old `permissions.go` bitmask code is dead but should be deleted to avoid confusion with the `auth.Role` type it defines

## Dependencies

- US-005 (auth store update) should be completed before US-006 through US-010 (frontend pages)
- US-011 (frontend cleanup) can be done alongside US-005 or US-006
- US-001 (middleware wiring) and US-002 (space restrictions) should be completed before US-003 (SOP scoping)
- US-004 (backend cleanup) is independent and can be done anytime
- US-012 and US-013 (CLI) are independent of frontend work
- US-007 (password change page) should be done before US-008 (onboarding) since the auth flow goes login -> password change -> onboarding

## Success Metrics

- All backend routes enforce proper authentication and authorization (no unauthenticated access to protected resources)
- Admin routes are accessible only to admin users
- Frontend login flow works end-to-end: login -> password change (if required) -> onboarding (if no spaces) -> main app
- Space switching works and API requests are scoped to the active space
- CLI `nori login` authenticates and stores credentials successfully
- Zero cross-space data leakage in SOP queries
- All quality gate commands pass (`go test ./...`, `go vet ./...`, `npm run build`, `npm run check`)

## Open Questions

- Should the onboarding flow include space templates (woodworking, sales, etc.) or just a name field for v1?
- Should the CLI cobra command structure live in `server/cmd/` or at the repo root?
- Should we add E2E tests for the full auth flow (login -> password change -> space creation) or defer that?