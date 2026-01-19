-- Remove dev team members
DELETE FROM team_members WHERE team_id = '00000000-0000-0000-0000-000000000001';

-- Remove dev users
DELETE FROM users WHERE oidc_issuer = 'dev-mode';

-- Remove dev team
DELETE FROM teams WHERE id = '00000000-0000-0000-0000-000000000001';
