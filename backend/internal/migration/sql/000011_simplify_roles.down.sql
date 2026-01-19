-- Revert to original role enum (admin, facilitator, participant)
-- Only runs if the simplified roles are in place (member exists in enum)

DO $$
BEGIN
    -- Check if 'member' exists in the current user_role enum (meaning up migration was applied)
    IF EXISTS (
        SELECT 1 FROM pg_enum
        WHERE enumtypid = 'user_role'::regtype
        AND enumlabel = 'member'
    ) THEN
        -- Drop default value first
        ALTER TABLE team_members ALTER COLUMN role DROP DEFAULT;

        -- Create the old enum type
        CREATE TYPE user_role_old AS ENUM ('admin', 'facilitator', 'participant');

        -- Convert column to text first to allow the transformation
        ALTER TABLE team_members ALTER COLUMN role TYPE TEXT;

        -- Update existing data to use old role names
        UPDATE team_members SET role = 'participant' WHERE role = 'member';

        -- Update the column to use the old type
        ALTER TABLE team_members
            ALTER COLUMN role TYPE user_role_old
            USING role::user_role_old;

        -- Drop the new enum and rename the old one
        DROP TYPE user_role;
        ALTER TYPE user_role_old RENAME TO user_role;

        -- Restore default value with old enum
        ALTER TABLE team_members ALTER COLUMN role SET DEFAULT 'participant'::user_role;
    END IF;
END $$;
