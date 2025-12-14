-- Cleanup script to delete all users except admin@musicalaideas.com
-- Run this on your production database

BEGIN;

-- First, let's see what we're going to delete (informational)
DO $$
DECLARE
    user_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO user_count FROM users WHERE email != 'admin@musicalaideas.com';
    RAISE NOTICE 'Will delete % user(s)', user_count;
END $$;

-- Get the list of user IDs to delete (excluding admin@musicalaideas.com)
CREATE TEMP TABLE users_to_delete AS
SELECT id FROM users WHERE email != 'admin@musicalaideas.com';

-- Delete email verification tokens for these users
DELETE FROM email_verification_tokens
WHERE user_id IN (SELECT id FROM users_to_delete);

-- Delete user credits
DELETE FROM user_credits
WHERE user_id IN (SELECT id FROM users_to_delete);

-- Delete usage logs
DELETE FROM usage_logs
WHERE user_id IN (SELECT id FROM users_to_delete);

-- Update invitation codes created by these users (nullify the reference)
UPDATE invitation_codes
SET created_by_id = NULL
WHERE created_by_id IN (SELECT id FROM users_to_delete);

-- Update invitation codes used by these users (nullify the reference)
UPDATE invitation_codes
SET used_by_id = NULL
WHERE used_by_id IN (SELECT id FROM users_to_delete);

-- Delete composition plans (this should cascade to generated_sequences if FK is set up)
DELETE FROM composition_plans
WHERE user_id IN (SELECT id FROM users_to_delete);

-- Delete generated sequences explicitly (in case cascade isn't set up)
-- First, get all plan IDs that belong to users being deleted
DELETE FROM generated_sequences
WHERE plan_id IN (
    SELECT id FROM composition_plans WHERE user_id IN (SELECT id FROM users_to_delete)
);

-- Finally, delete the users themselves
DELETE FROM users
WHERE id IN (SELECT id FROM users_to_delete);

-- Clean up temp table
DROP TABLE users_to_delete;

-- Show remaining users
SELECT id, email, role, is_active, email_verified, created_at
FROM users
ORDER BY id;

-- If everything looks good, commit. Otherwise, you can rollback.
-- COMMIT;
-- ROLLBACK;

-- For now, leaving it uncommitted so you can review
END;
