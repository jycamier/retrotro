-- Drop triggers
DROP TRIGGER IF EXISTS update_recurring_retros_updated_at ON recurring_retros;
DROP TRIGGER IF EXISTS update_integrations_updated_at ON integrations;
DROP TRIGGER IF EXISTS update_action_items_updated_at ON action_items;
DROP TRIGGER IF EXISTS update_items_updated_at ON items;
DROP TRIGGER IF EXISTS update_retrospectives_updated_at ON retrospectives;
DROP TRIGGER IF EXISTS update_teams_updated_at ON teams;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS retro_phase_history;
DROP TABLE IF EXISTS team_health_snapshots;
DROP TABLE IF EXISTS recurring_retros;
DROP TABLE IF EXISTS integrations;
DROP TABLE IF EXISTS action_items;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS retro_participants;
DROP TABLE IF EXISTS retrospectives;
DROP TABLE IF EXISTS template_phase_timers;
DROP TABLE IF EXISTS templates;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;

-- Drop custom types
DROP TYPE IF EXISTS retro_status;
DROP TYPE IF EXISTS retro_phase;
DROP TYPE IF EXISTS user_role;

-- Drop extension
DROP EXTENSION IF EXISTS "uuid-ossp";
