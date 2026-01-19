-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create custom types
CREATE TYPE user_role AS ENUM ('admin', 'facilitator', 'participant');
CREATE TYPE retro_phase AS ENUM ('brainstorm', 'group', 'vote', 'discuss', 'action');
CREATE TYPE retro_status AS ENUM ('draft', 'active', 'completed', 'archived');

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    oidc_subject VARCHAR(255) NOT NULL,
    oidc_issuer VARCHAR(500) NOT NULL,
    is_admin BOOLEAN DEFAULT false,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT users_oidc_unique UNIQUE (oidc_subject, oidc_issuer)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oidc ON users(oidc_subject, oidc_issuer);

-- Teams table with OIDC JIT support
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    oidc_group_id VARCHAR(255) UNIQUE,
    is_oidc_managed BOOLEAN DEFAULT false,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_teams_slug ON teams(slug);
CREATE INDEX idx_teams_oidc_group ON teams(oidc_group_id) WHERE oidc_group_id IS NOT NULL;

-- Team members table
CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role user_role NOT NULL DEFAULT 'participant',
    is_oidc_synced BOOLEAN DEFAULT false,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT team_members_unique UNIQUE (team_id, user_id)
);

CREATE INDEX idx_team_members_team ON team_members(team_id);
CREATE INDEX idx_team_members_user ON team_members(user_id);

-- Templates table
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    columns JSONB NOT NULL DEFAULT '[]',
    is_built_in BOOLEAN DEFAULT false,
    team_id UUID REFERENCES teams(id) ON DELETE CASCADE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_templates_team ON templates(team_id);
CREATE INDEX idx_templates_built_in ON templates(is_built_in);

-- Template phase timers
CREATE TABLE template_phase_timers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    phase retro_phase NOT NULL,
    duration_seconds INTEGER NOT NULL DEFAULT 300,
    is_optional BOOLEAN DEFAULT false,
    CONSTRAINT template_phase_unique UNIQUE (template_id, phase)
);

-- Retrospectives table
CREATE TABLE retrospectives (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE RESTRICT,
    facilitator_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status retro_status NOT NULL DEFAULT 'draft',
    current_phase retro_phase DEFAULT 'brainstorm',
    max_votes_per_user INTEGER NOT NULL DEFAULT 5,
    anonymous_voting BOOLEAN DEFAULT true,
    timer_started_at TIMESTAMP WITH TIME ZONE,
    timer_duration_seconds INTEGER,
    timer_paused_at TIMESTAMP WITH TIME ZONE,
    timer_remaining_seconds INTEGER,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_retrospectives_team ON retrospectives(team_id);
CREATE INDEX idx_retrospectives_status ON retrospectives(status);
CREATE INDEX idx_retrospectives_facilitator ON retrospectives(facilitator_id);

-- Retro participants table
CREATE TABLE retro_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    retro_id UUID NOT NULL REFERENCES retrospectives(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_online BOOLEAN DEFAULT false,
    last_seen_at TIMESTAMP WITH TIME ZONE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT retro_participants_unique UNIQUE (retro_id, user_id)
);

CREATE INDEX idx_retro_participants_retro ON retro_participants(retro_id);

-- Items table
CREATE TABLE items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    retro_id UUID NOT NULL REFERENCES retrospectives(id) ON DELETE CASCADE,
    column_id VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    group_id UUID REFERENCES items(id) ON DELETE SET NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_items_retro ON items(retro_id);
CREATE INDEX idx_items_column ON items(retro_id, column_id);
CREATE INDEX idx_items_group ON items(group_id) WHERE group_id IS NOT NULL;

-- Votes table
CREATE TABLE votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT votes_unique UNIQUE (item_id, user_id)
);

CREATE INDEX idx_votes_item ON votes(item_id);
CREATE INDEX idx_votes_user ON votes(user_id);

-- Action items table
CREATE TABLE action_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    retro_id UUID NOT NULL REFERENCES retrospectives(id) ON DELETE CASCADE,
    item_id UUID REFERENCES items(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date DATE,
    is_completed BOOLEAN DEFAULT false,
    completed_at TIMESTAMP WITH TIME ZONE,
    priority INTEGER DEFAULT 0,
    external_id VARCHAR(255),
    external_url TEXT,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_action_items_retro ON action_items(retro_id);
CREATE INDEX idx_action_items_assignee ON action_items(assignee_id);
CREATE INDEX idx_action_items_completed ON action_items(is_completed);

-- Integrations table
CREATE TABLE integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    config TEXT NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_integrations_team ON integrations(team_id);
CREATE INDEX idx_integrations_type ON integrations(type);

-- Recurring retros table
CREATE TABLE recurring_retros (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    facilitator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_enabled BOOLEAN DEFAULT true,
    next_scheduled_at TIMESTAMP WITH TIME ZONE,
    last_run_at TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_recurring_retros_team ON recurring_retros(team_id);
CREATE INDEX idx_recurring_retros_next ON recurring_retros(next_scheduled_at) WHERE is_enabled = true;

-- Team health snapshots table
CREATE TABLE team_health_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    retro_id UUID REFERENCES retrospectives(id) ON DELETE SET NULL,
    period VARCHAR(50) NOT NULL,
    metrics JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_health_snapshots_team ON team_health_snapshots(team_id);
CREATE INDEX idx_health_snapshots_period ON team_health_snapshots(team_id, period);

-- Retro phase history table
CREATE TABLE retro_phase_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    retro_id UUID NOT NULL REFERENCES retrospectives(id) ON DELETE CASCADE,
    phase retro_phase NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ended_at TIMESTAMP WITH TIME ZONE,
    actual_duration_seconds INTEGER,
    planned_duration_seconds INTEGER NOT NULL
);

CREATE INDEX idx_phase_history_retro ON retro_phase_history(retro_id);

-- Sessions table for refresh tokens
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address VARCHAR(45),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- Insert built-in templates
INSERT INTO templates (id, name, description, columns, is_built_in) VALUES
(uuid_generate_v4(), 'Start/Stop/Continue', 'Classic 3-column retrospective format', '[
    {"id": "start", "name": "Start", "description": "Things we should start doing", "color": "#22c55e", "icon": "play", "order": 0},
    {"id": "stop", "name": "Stop", "description": "Things we should stop doing", "color": "#ef4444", "icon": "stop", "order": 1},
    {"id": "continue", "name": "Continue", "description": "Things we should keep doing", "color": "#3b82f6", "icon": "repeat", "order": 2}
]', true),

(uuid_generate_v4(), 'Mad/Sad/Glad', 'Emotion-focused retrospective', '[
    {"id": "mad", "name": "Mad", "description": "Things that frustrated us", "color": "#ef4444", "icon": "angry", "order": 0},
    {"id": "sad", "name": "Sad", "description": "Things that disappointed us", "color": "#6366f1", "icon": "frown", "order": 1},
    {"id": "glad", "name": "Glad", "description": "Things that made us happy", "color": "#22c55e", "icon": "smile", "order": 2}
]', true),

(uuid_generate_v4(), '4Ls', 'Liked, Learned, Lacked, Longed For', '[
    {"id": "liked", "name": "Liked", "description": "Things we enjoyed", "color": "#22c55e", "icon": "heart", "order": 0},
    {"id": "learned", "name": "Learned", "description": "Things we discovered", "color": "#3b82f6", "icon": "lightbulb", "order": 1},
    {"id": "lacked", "name": "Lacked", "description": "Things we missed", "color": "#f59e0b", "icon": "x-circle", "order": 2},
    {"id": "longed-for", "name": "Longed For", "description": "Things we wished for", "color": "#8b5cf6", "icon": "star", "order": 3}
]', true),

(uuid_generate_v4(), 'Sailboat', 'Visual metaphor retrospective', '[
    {"id": "wind", "name": "Wind", "description": "What pushed us forward", "color": "#22c55e", "icon": "wind", "order": 0},
    {"id": "anchor", "name": "Anchor", "description": "What held us back", "color": "#ef4444", "icon": "anchor", "order": 1},
    {"id": "rocks", "name": "Rocks", "description": "Risks and obstacles ahead", "color": "#f59e0b", "icon": "alert-triangle", "order": 2},
    {"id": "island", "name": "Island", "description": "Our goals and vision", "color": "#3b82f6", "icon": "flag", "order": 3}
]', true);

-- Insert default phase timers for built-in templates
INSERT INTO template_phase_timers (template_id, phase, duration_seconds)
SELECT t.id, phase, duration
FROM templates t
CROSS JOIN (
    VALUES
        ('brainstorm'::retro_phase, 300),
        ('group'::retro_phase, 180),
        ('vote'::retro_phase, 180),
        ('discuss'::retro_phase, 900),
        ('action'::retro_phase, 300)
) AS phases(phase, duration)
WHERE t.is_built_in = true;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_retrospectives_updated_at BEFORE UPDATE ON retrospectives FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_items_updated_at BEFORE UPDATE ON items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_action_items_updated_at BEFORE UPDATE ON action_items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_integrations_updated_at BEFORE UPDATE ON integrations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_recurring_retros_updated_at BEFORE UPDATE ON recurring_retros FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
