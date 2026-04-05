# Auth and Tenancy

## Who

- **Admins**: Create and manage the Nori instance, create users, configure
  Spaces, manage all settings. Full access to everything.
- **Users**: Operate within Spaces they've been granted access to. Can create
  and execute work, manage SOPs, view analytics — everything except admin
  settings.

## What

Authentication and authorization system for a self-hosted Nori instance.
Users belong to a single Account (one per instance for v1), operate within
Spaces, and have roles that control what they can do. Designed to evolve into
multi-tenant SaaS if needed.

## Where

- Backend: `server/internal/auth/`, `server/internal/models/`,
  `server/internal/middleware/`, `server/internal/services/`
- Frontend: Login page, password change flow, Space selector, admin settings
- CLI: `nori login`, token storage
- Config: Environment variables for initial admin seed

## Why

Even a small shop needs access control. The shop owner needs full
configuration access, while operators on the floor need a fast,
minimal-friction experience without being overwhelmed by admin features.

The Account model is retained even though v1 is single-tenant because Nori
may evolve into a hosted SaaS product. Building the hierarchy now avoids a
painful migration later.

Spaces separate different areas of work (Shop, Sales, Marketing, Web Dev),
each with their own ticket types, statuses, and workflows. Admins see all
Spaces; users are explicitly granted access to specific Spaces.

## How

### Entity Hierarchy

```
Account (one per instance in v1, future: one per business)
  ├── User (belongs to the Account, has a role: admin or user)
  └── Space (isolated workspace within the Account)
        └── SpaceMember (grants a User access to a Space)
```

### Roles

Two roles, applied at the Account level:

| Role | Access |
|------|--------|
| `admin` | Full access. User management, Space configuration, account settings, plus everything a user can do. |
| `user` | Access to Spaces they're a member of. Can create/edit tickets, execute work, manage SOPs, view analytics. Cannot access admin settings. |

Role is stored on the `User` model (or `UserAccount` join table if
multi-tenancy is enabled). No per-resource bitmask permissions — role
determines access, checked in middleware.

### Space Access

- **Admins** have implicit access to all Spaces. No SpaceMember record needed.
- **Users** are granted access to specific Spaces via the `SpaceMember` join
  table. If a user has no SpaceMember records, they see nothing.

```
SpaceMember {
  ID        uuid
  UserID    uuid → User
  SpaceID   uuid → Space
  CreatedAt timestamp
}
```

No per-Space role differentiation in v1. The user's Account-level role
(admin/user) determines their permissions everywhere.

### Authentication

**Email + password only** for v1. No OAuth/SSO.

- Passwords hashed with bcrypt (existing implementation).
- JWT-based authentication carried forward from existing code.
- Secret key loaded from environment variable (replace current hardcoded key).

### Session Management

Long-lived sessions, modeled after Atlassian's self-hosted products:

- On login, issue a JWT with **30-day expiry**.
- Token stored as an HTTP-only cookie (web) or in a local config file (CLI).
- No refresh token mechanism for v1 — when the token expires, user logs in
  again.
- Middleware validates the JWT on every request and extracts user identity.

### API Keys

Account-level service keys for headless access (CLI, MCP server, sensors):

```
APIKey {
  ID          uuid
  AccountID   uuid → Account
  Name        string         -- human-readable label (e.g., "MCP Server", "Shop Tablet")
  KeyHash     string         -- bcrypt hash of the key (key shown once on creation)
  LastUsedAt  *timestamp
  ExpiresAt   *timestamp     -- optional expiry
  IsActive    bool
  CreatedAt   timestamp
  CreatedByID uuid → User    -- which admin created it
}
```

- The raw key is shown once at creation time and never stored.
- API key is sent via `Authorization: Bearer nori_...` header.
- Middleware detects the `nori_` prefix to distinguish API keys from JWTs.
- API keys act with admin-level access (they're Account-level service keys).
- Admins can deactivate or delete keys.

### User Management

Admins create users directly (no invite-by-email flow):

1. Admin fills in email, name, and a temporary password.
2. User's `MustChangePassword` flag is set to `true`.
3. On first login, the user is redirected to a password change screen.
4. After setting a new password, the flag is cleared and normal access begins.

### First Boot / Setup

Instance setup is seeded from environment variables:

```env
NORI_ADMIN_EMAIL=owner@shop.com
NORI_ADMIN_PASSWORD=changeme
NORI_ACCOUNT_NAME=My Woodshop
```

On first boot (no Account exists in the database):

1. Create the Account with the configured name.
2. Create the admin User with the provided credentials.
3. Set `MustChangePassword = true` on the admin user.
4. **Do not create any Spaces.**

On first login, the admin sees an onboarding prompt to create their first
Space by choosing a template (woodworking shop, sales, etc.). Every Space is
intentional — no pre-created defaults.

### Data Isolation

All queries are scoped by Space. The middleware extracts the active Space
from the request context (header or JWT claim). No cross-space data leakage.

The active Space is determined by:
1. `X-Space-ID` header (explicit per-request), or
2. User's most recent Space (from `RecentSpaces` on the User model)

### UI Behavior

- **Login page**: Email + password form. No registration — admins create users.
- **First login**: Forced password change screen.
- **Admin first login**: Onboarding prompt to create first Space with template.
- **Space selector**: Header dropdown for switching between Spaces the user
  has access to (admins see all, users see their SpaceMember Spaces).
- **Role-based UI**: Users see all features within their Spaces. Admins
  additionally see: user management, Space configuration, API key management,
  account settings.

### Prior Art (Existing Code to Refactor)

Models to keep and update:
- `User` — add `MustChangePassword bool`, keep `GlobalRole` (rename to just
  `Role` with values `admin`/`user`), keep `RecentSpaces`
- `Account` — simplify: drop billing/contact fields for v1, keep `Name`,
  `Plan`, `CreatedByUserID`
- `UserAccount` — keep for future multi-tenancy, but v1 only has one Account
- `Space` — keep as-is

New models:
- `SpaceMember` — join table granting users access to specific Spaces
- `APIKey` — Account-level service keys for headless access

Code to refactor:
- `auth.service.go` — move JWT secret to env var, extend token expiry to 30
  days, add API key authentication path, add password change flow
- `auth.middleware.go` — consolidate `c.Locals` keys (currently uses both
  "authDTO" and "user"), add API key detection (`nori_` prefix), add Space
  scoping from header/recent
- `permissions.go` — replace bitmask system with simple role check
  (admin vs user)
- Remove self-registration — admins create users, no public signup

## Open Questions

None remaining for v1. Future considerations:
- OAuth/SSO support for SaaS multi-tenancy
- Per-Space roles (owner/manager/operator) if needed for larger shops
- Invite-by-email flow if email infrastructure is available
- API key scoping per-Space (currently Account-level)
