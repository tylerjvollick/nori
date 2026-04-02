# Auth and Tenancy — Implementation Checklist

Each item is a small, committable unit of work. Work top-down.

## 1. Environment and Configuration

- [x] **Add config struct and env loading for auth settings**
  - Create `server/internal/config/` package (or extend existing)
  - Load `NORI_JWT_SECRET`, `NORI_ADMIN_EMAIL`, `NORI_ADMIN_PASSWORD`,
    `NORI_ACCOUNT_NAME` from environment
  - Validate required vars are set on startup
  - Replace hardcoded `"your-secret-key"` in auth service with config value

## 2. Model Changes

- [x] **Update User model**
  - Add `MustChangePassword bool` field (default `true` for new users)
  - Rename `GlobalRole` to `Role` with values `admin` / `user` (drop `viewer`)
  - Add GORM migration for the new/renamed fields

- [x] **Simplify Account model**
  - Drop billing/contact fields for v1 (or mark as nullable/unused)
  - Keep `Name`, `Plan`, `CreatedByUserID`
  - Add migration

- [x] **Create SpaceMember model**
  - `ID`, `UserID`, `SpaceID`, `CreatedAt`
  - Unique constraint on `(UserID, SpaceID)`
  - Add GORM migration

- [x] **Create APIKey model**
  - `ID`, `AccountID`, `Name`, `KeyHash`, `LastUsedAt`, `ExpiresAt`,
    `IsActive`, `CreatedAt`, `CreatedByID`
  - Add GORM migration

## 3. Repositories

- [x] **Create SpaceMember repository**
  - `Create`, `Delete`, `GetByUserAndSpace`, `GetByUser` (list Spaces for a
    user), `GetBySpace` (list members of a Space)

- [x] **Create APIKey repository**
  - `Create`, `GetByKeyHash`, `GetByAccount`, `Deactivate`, `Delete`,
    `UpdateLastUsed`

- [x] **Update User repository**
  - Add `GetByEmail` if not already present
  - Add `UpdatePassword`, `ClearMustChangePassword`

## 4. Auth Service Refactor

- [x] **Refactor JWT handling**
  - Use secret from config (not hardcoded)
  - Extend token expiry to 30 days
  - Set token as HTTP-only cookie (web) in addition to response body
  - Add `ActiveSpaceID` claim to JWT (optional, from `X-Space-ID` or
    RecentSpaces)

- [x] **Add API key authentication path**
  - Generate keys with `nori_` prefix + random bytes
  - Hash with SHA256 before storing (deterministic hash for lookup)
  - On auth, detect `nori_` prefix → look up by hash → validate active/expiry
  - Update `LastUsedAt` on successful auth

- [x] **Add password change flow**
  - `ChangePassword(userID, oldPassword, newPassword)` method
  - Verify old password, hash new, clear `MustChangePassword` flag
  - Return new JWT after password change

- [x] **Remove self-registration**
  - Remove or disable the public registration endpoint
  - Only admins can create users (via admin API)

## 5. Middleware Refactor

- [x] **Consolidate auth middleware**
  - Fix `c.Locals` key inconsistency ("authDTO" vs "user") — use one key
  - Support both JWT cookies and Authorization header
  - Add API key detection (`nori_` prefix) branch
  - Extract active Space from `X-Space-ID` header or user's RecentSpaces

- [x] **Replace bitmask permissions with role check**
  - Remove `permissions.go` bitmask system
  - Create `RequireAdmin()` middleware — checks `user.Role == admin`
  - Create `RequireSpaceAccess(spaceID)` middleware — admins pass
    automatically, users must have a SpaceMember record

- [x] **Add MustChangePassword guard**
  - Middleware that checks `user.MustChangePassword`
  - If true, reject all requests except the password change endpoint
  - Return a specific error code so the frontend knows to redirect

## 6. Admin User Management API

- [x] **Create admin user management endpoints**
  - `POST /api/admin/users` — create user (email, name, temp password)
  - `GET /api/admin/users` — list all users in the Account
  - `PUT /api/admin/users/:id` — update user (name, role)
  - `DELETE /api/admin/users/:id` — deactivate/delete user
  - All protected by `RequireAdmin()` middleware

- [x] **Create Space membership endpoints**
  - `POST /api/admin/spaces/:id/members` — add user to Space
  - `DELETE /api/admin/spaces/:id/members/:userId` — remove user from Space
  - `GET /api/admin/spaces/:id/members` — list Space members

- [ ] **Create API key management endpoints**
  - `POST /api/admin/api-keys` — create key (return raw key once)
  - `GET /api/admin/api-keys` — list keys (name, last used, active status)
  - `DELETE /api/admin/api-keys/:id` — deactivate/delete key

## 7. First Boot / Seed

- [ ] **Implement first-boot seed logic**
  - On startup, check if any Account exists in the database
  - If none: create Account, create admin User with env credentials,
    set `MustChangePassword = true`
  - If Account exists: skip seeding (idempotent)
  - Log the seeding action to stdout

## 8. Login and Session Endpoints

- [ ] **Refactor login endpoint**
  - `POST /api/auth/login` — validate credentials, return JWT as HTTP-only
    cookie + JSON body
  - If `MustChangePassword`, return a flag in the response so the client
    knows to redirect

- [ ] **Add password change endpoint**
  - `POST /api/auth/change-password` — requires current password + new
    password. Clears `MustChangePassword`, returns new JWT.

- [ ] **Add session info endpoint**
  - `GET /api/auth/me` — return current user, role, active Space, list of
    accessible Spaces

- [ ] **Add logout endpoint**
  - `POST /api/auth/logout` — clear the HTTP-only cookie

## 9. Space Scoping

- [ ] **Add Space scoping to existing queries**
  - Ensure all ticket, SOP, station, etc. queries filter by the active
    SpaceID from middleware context
  - Verify no cross-space data leakage in existing repositories

- [ ] **Update Space service for onboarding**
  - `POST /api/spaces` — create Space with template (existing template
    service)
  - Ensure admins can create Spaces, users cannot
  - On creation, admin is not added to SpaceMember (they have implicit
    access)

## 10. Frontend (Web)

- [ ] **Login page**
  - Email + password form
  - Handle `MustChangePassword` flag → redirect to password change

- [ ] **Password change page**
  - Old password + new password + confirm
  - Redirect to main app after success

- [ ] **Onboarding: create first Space**
  - Shown when admin logs in and no Spaces exist
  - Space name + template selector
  - Creates Space and redirects to it

- [ ] **Space selector**
  - Header dropdown showing accessible Spaces
  - Switching sets `X-Space-ID` header on subsequent requests
  - Updates user's `RecentSpaces`

- [ ] **Admin settings pages**
  - User management (list, create, edit role, deactivate)
  - API key management (list, create, revoke)
  - Space member management (add/remove users per Space)
  - Only visible to admin role

## 11. CLI Auth

- [ ] **Implement `nori login` command**
  - Prompt for server URL + email + password
  - Call login endpoint, store JWT in `~/.config/nori/credentials`
  - Handle `MustChangePassword` (prompt for new password inline)

- [ ] **Add token refresh/re-login to CLI**
  - On 401, prompt user to re-authenticate
  - Support API key auth as alternative: `nori config set api-key nori_...`
