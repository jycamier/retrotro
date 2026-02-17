-- Add status column to action_items
ALTER TABLE action_items ADD COLUMN status TEXT NOT NULL DEFAULT 'todo';

-- Migrate existing data
UPDATE action_items SET status = 'done' WHERE is_completed = true;
UPDATE action_items SET status = 'in_progress' WHERE is_completed = false AND due_date IS NOT NULL;

-- Index for status queries
CREATE INDEX idx_action_items_status ON action_items(status);
