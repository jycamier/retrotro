DROP INDEX IF EXISTS idx_lc_topic_history_retro;
DROP TABLE IF EXISTS lc_topic_history;

ALTER TABLE retrospectives DROP COLUMN IF EXISTS lc_topic_timebox_seconds;
ALTER TABLE retrospectives DROP COLUMN IF EXISTS lc_current_topic_id;
ALTER TABLE retrospectives DROP COLUMN IF EXISTS session_type;

-- Note: PostgreSQL does not support removing values from enums.
-- The 'propose' value will remain in the retro_phase enum.
