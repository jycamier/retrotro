-- Simplify team member roles: remove 'facilitator' role, keep only 'admin' and 'member'
-- Rename 'participant' to 'member' for clarity

-- Drop default value first (it references the old enum)
ALTER TABLE team_members ALTER COLUMN role DROP DEFAULT;

-- Convert column to TEXT first to allow the transformation
ALTER TABLE team_members ALTER COLUMN role TYPE TEXT;

-- Update existing data to use new role names
UPDATE team_members SET role = 'member' WHERE role = 'facilitator';
UPDATE team_members SET role = 'member' WHERE role = 'participant';

-- Create new enum type
CREATE TYPE user_role_new AS ENUM ('admin', 'member');

-- Update the column to use the new type
ALTER TABLE team_members
    ALTER COLUMN role TYPE user_role_new
    USING role::user_role_new;

-- Drop the old enum and rename the new one
DROP TYPE user_role;
ALTER TYPE user_role_new RENAME TO user_role;

-- Restore default value with new enum
ALTER TABLE team_members ALTER COLUMN role SET DEFAULT 'member'::user_role;
