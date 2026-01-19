-- Migration: Allow multiple votes per item
-- Changes the voting system from 1 vote per item to N votes (configurable, default 3)

-- Add max_votes_per_item column to retrospectives
ALTER TABLE retrospectives
ADD COLUMN max_votes_per_item INT NOT NULL DEFAULT 3;

-- Drop the unique constraint on votes to allow multiple votes
ALTER TABLE votes DROP CONSTRAINT votes_unique;

-- Add index for performance on counting votes per user per item
CREATE INDEX idx_votes_item_user ON votes(item_id, user_id);
