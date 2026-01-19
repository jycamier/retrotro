-- Dev users seed (only runs in dev mode via docker-entrypoint-initdb.d)
-- This is idempotent - safe to run multiple times

-- Create Dev Team
INSERT INTO teams (id, name, slug, description, is_oidc_managed)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Dev Team',
    'dev-team',
    'Development team for testing',
    false
)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description;

-- Create dev users with ON CONFLICT to handle existing users
INSERT INTO users (id, email, display_name, oidc_subject, oidc_issuer, is_admin)
VALUES
    ('00000000-0000-0000-0001-000000000001', 'admin@retrotro.dev', 'Dev Admin', 'dev-admin@retrotro.dev', 'dev-mode', true),
    ('00000000-0000-0000-0001-000000000002', 'manager@retrotro.dev', 'Team Manager', 'dev-manager@retrotro.dev', 'dev-mode', false),
    ('00000000-0000-0000-0001-000000000003', 'facilitator@retrotro.dev', 'Facilitateur', 'dev-facilitator@retrotro.dev', 'dev-mode', false),
    ('00000000-0000-0000-0001-000000000004', 'user1@retrotro.dev', 'User One', 'dev-user1@retrotro.dev', 'dev-mode', false),
    ('00000000-0000-0000-0001-000000000005', 'user2@retrotro.dev', 'User Two', 'dev-user2@retrotro.dev', 'dev-mode', false),
    ('00000000-0000-0000-0001-000000000006', 'user3@retrotro.dev', 'User Three', 'dev-user3@retrotro.dev', 'dev-mode', false)
ON CONFLICT (email) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    oidc_subject = EXCLUDED.oidc_subject,
    oidc_issuer = EXCLUDED.oidc_issuer,
    is_admin = EXCLUDED.is_admin;

-- Add all dev users to the dev team with their roles
INSERT INTO team_members (team_id, user_id, role)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0001-000000000001', 'admin'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0001-000000000002', 'admin'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0001-000000000003', 'facilitator'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0001-000000000004', 'participant'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0001-000000000005', 'participant'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0001-000000000006', 'participant')
ON CONFLICT (team_id, user_id) DO UPDATE SET
    role = EXCLUDED.role;
