-- Migration: Add retro_attendees table for tracking attendance
-- This table records who was present when a retrospective started (waiting -> icebreaker transition)

CREATE TABLE retro_attendees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    retrospective_id UUID NOT NULL REFERENCES retrospectives(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attended BOOLEAN NOT NULL DEFAULT false,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(retrospective_id, user_id)
);

CREATE INDEX idx_retro_attendees_retro ON retro_attendees(retrospective_id);
CREATE INDEX idx_retro_attendees_user ON retro_attendees(user_id);

COMMENT ON TABLE retro_attendees IS 'Records attendance for each retrospective - who was present when the retro started';
COMMENT ON COLUMN retro_attendees.attended IS 'Whether the user was connected when transitioning from waiting to icebreaker phase';
