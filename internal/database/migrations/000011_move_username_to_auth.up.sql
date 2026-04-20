-- Move username from users table to auths table

-- Step 1: Add username column to auths table
ALTER TABLE auths ADD COLUMN username VARCHAR(64) NOT NULL DEFAULT '';

-- Step 2: Create unique index on auths.username
CREATE UNIQUE INDEX idx_auths_username ON auths(LOWER(username));

-- Step 3: Migrate existing usernames from users to auths
UPDATE auths SET username = (SELECT username FROM users WHERE users.id = auths.user_id);

-- Step 4: Remove username column from users table
ALTER TABLE users DROP COLUMN username;

-- Step 5: Drop the old unique index on users.username (if exists)
DROP INDEX IF EXISTS idx_users_username_lower;
