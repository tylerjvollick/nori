# Auth and Tenancy

## Who

- **Shop owners**: Create accounts, manage spaces, invite team members.
- **Floor managers**: Manage SOPs, assign jobs, view analytics within a space.
- **Operators**: Execute jobs, check in/out at stations, capture SOP notes.

## What

Multi-tenant authentication and authorization system. Users belong to Accounts,
operate within Spaces, and have roles that control what they can do.

## Where

- Backend: `server/internal/auth/`, `server/internal/models/`, `server/internal/middleware/`
- Frontend: Login/Register pages, space selector, role-based UI
- CLI: `nori login`, token management

## Why

Nori is designed for small shops but built to support multiple shops
(multi-tenancy). A single Nori instance can serve multiple businesses, each
with their own data isolation. Within a business, Spaces allow separation
(e.g., "Main Shop" vs. "Finishing Room") while sharing the same account and
user pool.

Roles matter because a shop owner needs analytics and configuration access,
while an operator on the floor needs fast, minimal-friction job execution
without being overwhelmed by admin features.

## How

### Account Hierarchy

```
Account (billing entity — one per business)
  ├── UserAccount (membership with role)
  │     └── User (can belong to multiple accounts)
  └── Space (isolated workspace)
        └── SpaceMember (user's role within this space)
```

### Roles

**Account-level roles** (existing: admin, user):
- `admin` — Full account management, billing, invite users
- `user` — Access spaces they're a member of

**Space-level roles** (new):
- `owner` — Full control over the space: configure stations, manage SOPs,
  view all analytics, manage members
- `manager` — Create/edit SOPs, assign jobs, view analytics, but cannot
  configure stations or manage space membership
- `operator` — Execute jobs, check in/out, add deviation notes to SOPs.
  Cannot create or edit SOP templates directly.

### Authentication Flow

Carried forward from existing implementation:
1. User registers with email + password
2. Account is created automatically on first registration
3. Default Space is created within the Account
4. JWT-based session tokens
5. Middleware validates token on every API request

Enhancements for v1:
- CLI login flow: `nori login` → prompts for server URL + credentials → stores
  token locally
- MCP server reuses the same JWT mechanism
- API key support for headless/automation use cases (CLI, sensors)

### Data Isolation

All queries are scoped by Space. The middleware extracts the active Space from
the request context (header or JWT claim). No cross-space data leakage.

### Prior Art

Existing models to carry forward:
- `User` — has GlobalRole, email, password, DefaultAccountID, RecentSpaces
- `Account` — has Plan (trial/paid/enterprise), contact/billing info
- `UserAccount` — join table with account-level Role
- `Space` — has Name, AccountID, IsDefault

New model needed:
- `SpaceMember` — join table between User and Space with space-level role

### UI Behavior

- On login, user lands in their most recent Space (from RecentSpaces)
- Space switcher in the header for users with multiple spaces
- Role-based UI: operators see a simplified view focused on the job flow
  board and active job steps. Managers see SOP editing and analytics.
  Owners see configuration.

## Open Questions

- Should we support OAuth/SSO for v1, or is email+password sufficient?
  (Leaning toward email+password only for v1 — simpler for self-hosted.)
- Do we need an explicit "invite" flow, or can account admins just create
  user accounts directly? (Invite via email link is nicer UX but more
  infrastructure.)
- Should API keys be scoped to a Space or to an Account?
