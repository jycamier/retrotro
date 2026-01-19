# OIDC Authentication

Retrotro supports OpenID Connect (OIDC) authentication with Just-In-Time (JIT) provisioning of teams from identity provider groups.

## Quick Start with Keycloak

A pre-configured Keycloak instance is available for testing OIDC authentication.

### Start with OIDC

```bash
docker compose -f docker-compose.yml -f docker-compose.oidc.yml up
```

### Pre-configured Users

| Utilisateur | Email | Mot de passe | Groupes |
|-------------|-------|--------------|---------|
| alice | alice@retrotro.dev | `alice123` | admins, team-alpha |
| bob | bob@retrotro.dev | `bob123` | team-alpha |
| charlie | charlie@retrotro.dev | `charlie123` | team-beta |
| diana | diana@retrotro.dev | `diana123` | team-alpha, team-beta |
| eve | eve@retrotro.dev | `eve123` | team-beta |

**Groupes :**
- `admins` → Alice sera admin via JIT
- `team-alpha` → Alice (admin), Bob, Diana
- `team-beta` → Charlie, Diana, Eve

### Access Points

- **Application**: http://localhost:3000
- **Keycloak Admin**: http://localhost:8180 (admin/admin)
- **Backend API**: http://localhost:8081

## How JIT Provisioning Works

When a user logs in via OIDC:

1. **User Creation**: If the user doesn't exist, they are created automatically
2. **Group Extraction**: Groups are extracted from the `groups` claim in the token
3. **Team Creation**: For each group, a team is created if it doesn't exist
4. **Membership Sync**: The user is added to teams matching their groups
5. **Role Assignment**: Users in admin groups get the `admin` role, others get `member`

```
OIDC Token (groups: ["team-alpha", "admins"])
    ↓
JIT Provisioner
    ↓
Creates/Updates:
  - Team "team-alpha" (if new)
  - Team "admins" (if new)
  - User membership with appropriate roles
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OIDC_ISSUER_URL` | OIDC provider issuer URL | - |
| `OIDC_CLIENT_ID` | OAuth2 client ID | - |
| `OIDC_CLIENT_SECRET` | OAuth2 client secret | - |
| `OIDC_REDIRECT_URL` | Callback URL after authentication | http://localhost:8080/auth/callback |
| `OIDC_SCOPES` | Scopes to request | openid,profile,email,groups |

### JIT Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `OIDC_JIT_ENABLED` | Enable JIT provisioning | true |
| `OIDC_JIT_GROUPS_CLAIM` | Token claim containing groups | groups |
| `OIDC_JIT_GROUPS_PREFIX` | Prefix to strip from group names | - |
| `OIDC_JIT_DEFAULT_ROLE` | Default role for team members | member |
| `OIDC_JIT_ADMIN_GROUPS` | Groups that grant admin role (comma-separated) | - |
| `OIDC_JIT_SYNC_ON_LOGIN` | Sync memberships on each login | true |
| `OIDC_JIT_REMOVE_STALE_MEMBERS` | Remove user from teams not in their groups | false |

### Example Configuration

```bash
# .env
OIDC_ISSUER_URL=https://your-idp.com/realms/your-realm
OIDC_CLIENT_ID=retrotro
OIDC_CLIENT_SECRET=your-secret
OIDC_REDIRECT_URL=https://retrotro.example.com/auth/callback

OIDC_JIT_ENABLED=true
OIDC_JIT_GROUPS_CLAIM=groups
OIDC_JIT_ADMIN_GROUPS=retrotro-admins,platform-admins
OIDC_JIT_DEFAULT_ROLE=member
```

## Authentication Flow

```
┌─────────┐     ┌──────────┐     ┌─────────┐     ┌──────────┐
│ Browser │     │ Frontend │     │ Backend │     │   IdP    │
└────┬────┘     └────┬─────┘     └────┬────┘     └────┬─────┘
     │               │                │               │
     │  Click Login  │                │               │
     │──────────────>│                │               │
     │               │  GET /auth/login               │
     │               │───────────────>│               │
     │               │                │               │
     │               │  302 Redirect to IdP           │
     │<──────────────────────────────────────────────>│
     │               │                │               │
     │  User authenticates            │               │
     │<──────────────────────────────────────────────>│
     │               │                │               │
     │  Redirect with code            │               │
     │───────────────────────────────>│               │
     │               │                │  Exchange code│
     │               │                │──────────────>│
     │               │                │  ID Token     │
     │               │                │<──────────────│
     │               │                │               │
     │               │                │  JIT Provision│
     │               │                │  (create user,│
     │               │                │   sync teams) │
     │               │                │               │
     │               │  Set session   │               │
     │<──────────────────────────────-│               │
     │               │                │               │
```

## Configuring Your Identity Provider

### Keycloak

1. Create a new realm or use an existing one
2. Create an OpenID Connect client:
   - Client ID: `retrotro-app`
   - Client Protocol: `openid-connect`
   - Access Type: `confidential`
   - Valid Redirect URIs: `http://localhost:8081/auth/callback`
3. Add a "groups" client scope with a group membership mapper:
   - Mapper Type: `Group Membership`
   - Token Claim Name: `groups`
   - Full group path: `OFF`
4. Assign the groups scope to your client

### Azure AD / Entra ID

1. Register a new application
2. Configure redirect URIs
3. Add optional claim for groups:
   - Go to Token Configuration
   - Add groups claim
   - Select "Security groups" or "Groups assigned to the application"
4. Note: Azure uses `groups` claim by default with group IDs, you may want to configure group names

### Okta

1. Create an OIDC application
2. Configure redirect URIs
3. Add groups claim:
   - Go to Sign On → OpenID Connect ID Token
   - Add groups claim with filter

### Google Workspace

Google Workspace doesn't support custom groups claims natively. Consider using a proxy like Dex or Pomerium to add group information.

## Troubleshooting

### Groups not appearing in token

1. Verify the groups scope is requested
2. Check that the groups mapper is configured in your IdP
3. Enable debug logging and check the raw token claims:
   ```bash
   LOGGER_LEVEL=debug
   ```

### Teams not being created

1. Verify `OIDC_JIT_ENABLED=true`
2. Check that `OIDC_JIT_GROUPS_CLAIM` matches your IdP's claim name
3. If your groups have a prefix (e.g., `/team-alpha`), configure `OIDC_JIT_GROUPS_PREFIX=/`

### User always gets member role

1. Check `OIDC_JIT_ADMIN_GROUPS` is set correctly
2. Verify the group name matches exactly (case-sensitive)
3. Check the user is actually in the admin group in your IdP

## Security Considerations

1. **Always use HTTPS** in production
2. **Rotate client secrets** regularly
3. **Limit redirect URIs** to your actual domains
4. **Consider `OIDC_JIT_REMOVE_STALE_MEMBERS`** to automatically remove users from teams they no longer belong to

## Related Documentation

- [Configuration](./configuration.md) - General configuration options
- [API Reference](./api-reference.md) - Authentication endpoints
