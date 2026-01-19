-- Drop the index for multi-votes
DROP INDEX IF EXISTS idx_votes_item_user;

-- Restore the unique constraint (this will fail if duplicate votes exist)
ALTER TABLE votes ADD CONSTRAINT votes_unique UNIQUE (item_id, user_id);

-- Remove max_votes_per_item column
ALTER TABLE retrospectives DROP COLUMN IF EXISTS max_votes_per_item;
