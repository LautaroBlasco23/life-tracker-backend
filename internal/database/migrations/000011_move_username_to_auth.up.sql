-- Move username from users table to auth table

-- Step 1: Add username column to auth table
ALTER TABLE auth ADD COLUMN username VARCHAR(64) NOT NULL DEFAULT '';

-- Step 2: Create unique index on auth.username
CREATE UNIQUE INDEX idx_auth_username ON auth(LOWER(username));

-- Step 3: Migrate existing usernames from users to auth
UPDATE auth SET username = (SELECT username FROM users WHERE users.id = auth.user_id);

-- Step 4: Remove username column from users table
ALTER TABLE users DROP COLUMN username;

-- Step 5: Drop the old unique index on users.username (if exists)
DROP INDEX IF EXISTS idx_users_username;
