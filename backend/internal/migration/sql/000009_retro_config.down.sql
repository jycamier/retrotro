-- Remove configuration columns from retrospectives
ALTER TABLE retrospectives
DROP COLUMN IF EXISTS phase_timer_overrides,
DROP COLUMN IF EXISTS allow_vote_change,
DROP COLUMN IF EXISTS allow_item_edit,
DROP COLUMN IF EXISTS anonymous_items;
