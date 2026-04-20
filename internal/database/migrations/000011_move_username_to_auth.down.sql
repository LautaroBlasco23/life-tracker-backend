-- Revert: Move username back to users table

-- Step 1: Add username column back to users table
ALTER TABLE users ADD COLUMN username VARCHAR(64) NOT NULL DEFAULT '';

-- Step 2: Create unique index on users.username
CREATE UNIQUE INDEX idx_users_username_lower ON users(LOWER(username)) WHERE deleted_at IS NULL;

-- Step 3: Migrate usernames from auths back to users
UPDATE users SET username = (SELECT username FROM auths WHERE auths.user_id = users.id);

-- Step 4: Remove username column from auths table
ALTER TABLE auths DROP COLUMN username;

-- Step 5: Drop the unique index on auths.username
DROP INDEX IF EXISTS idx_auths_username;
