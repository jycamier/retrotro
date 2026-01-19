-- Remove roti_revealed column from retrospectives
ALTER TABLE retrospectives DROP COLUMN IF EXISTS roti_revealed;

-- Drop tables
DROP TABLE IF EXISTS roti_votes;
DROP TABLE IF EXISTS icebreaker_moods;

-- Drop mood_weather type
DROP TYPE IF EXISTS mood_weather;

-- Note: PostgreSQL does not support removing values from an enum type.
-- The 'icebreaker' and 'roti' values will remain in retro_phase enum.
-- To fully remove them, you would need to recreate the enum type.
