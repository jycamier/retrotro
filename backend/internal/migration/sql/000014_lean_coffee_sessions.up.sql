-- Add 'propose' phase to retro_phase enum
ALTER TYPE retro_phase ADD VALUE IF NOT EXISTS 'propose';

-- Add session_type to retrospectives (retro or lean_coffee)
ALTER TABLE retrospectives ADD COLUMN IF NOT EXISTS session_type VARCHAR(20) NOT NULL DEFAULT 'retro';

-- Lean Coffee specific fields: current topic tracking and timebox config
ALTER TABLE retrospectives ADD COLUMN IF NOT EXISTS lc_current_topic_id UUID REFERENCES items(id) ON DELETE SET NULL;
ALTER TABLE retrospectives ADD COLUMN IF NOT EXISTS lc_topic_timebox_seconds INTEGER DEFAULT 300;

-- Discussion history per topic (for wrapup and stats)
CREATE TABLE IF NOT EXISTS lc_topic_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    retro_id UUID NOT NULL REFERENCES retrospectives(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    discussion_order INTEGER NOT NULL,
    total_discussion_seconds INTEGER NOT NULL DEFAULT 0,
    extension_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ended_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT lc_topic_history_unique UNIQUE (retro_id, topic_id)
);

CREATE INDEX IF NOT EXISTS idx_lc_topic_history_retro ON lc_topic_history(retro_id);
