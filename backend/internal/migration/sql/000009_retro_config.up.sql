-- Migration: Add advanced configuration options to retrospectives

-- Add new configuration columns
ALTER TABLE retrospectives
ADD COLUMN anonymous_items BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN allow_item_edit BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN allow_vote_change BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN phase_timer_overrides JSONB;

-- Add comment for clarity
COMMENT ON COLUMN retrospectives.anonymous_items IS 'When true, item authors are hidden from other participants';
COMMENT ON COLUMN retrospectives.allow_item_edit IS 'When true, participants can edit their items after creation';
COMMENT ON COLUMN retrospectives.allow_vote_change IS 'When true, participants can remove their votes';
COMMENT ON COLUMN retrospectives.phase_timer_overrides IS 'JSON object mapping phase names to custom durations in seconds';
