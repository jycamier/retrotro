-- Performance indexes for statistics queries
CREATE INDEX IF NOT EXISTS idx_roti_votes_retro_rating ON roti_votes(retro_id, rating);
CREATE INDEX IF NOT EXISTS idx_icebreaker_moods_retro_mood ON icebreaker_moods(retro_id, mood);
CREATE INDEX IF NOT EXISTS idx_retrospectives_team_status ON retrospectives(team_id, status) WHERE status = 'completed';
CREATE INDEX IF NOT EXISTS idx_roti_votes_user ON roti_votes(user_id);
CREATE INDEX IF NOT EXISTS idx_icebreaker_moods_user ON icebreaker_moods(user_id);
