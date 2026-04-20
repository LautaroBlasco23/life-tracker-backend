-- Revert: Move username back to users table

-- Step 1: Add username column back to users table
ALTER TABLE users ADD COLUMN username VARCHAR(64) NOT NULL DEFAULT '';

-- Step 2: Create unique index on users.username
CREATE UNIQUE INDEX idx_users_username ON users(LOWER(username));

-- Step 3: Migrate usernames from auth back to users
UPDATE users SET username = (SELECT username FROM auth WHERE auth.user_id = users.id);

-- Step 4: Remove username column from auth table
ALTER TABLE auth DROP COLUMN username;

-- Step 5: Drop the unique index on auth.username
DROP INDEX IF EXISTS idx_auth_username;
