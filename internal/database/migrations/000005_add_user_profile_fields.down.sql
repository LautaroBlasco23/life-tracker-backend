DROP INDEX IF EXISTS idx_users_username_lower;
ALTER TABLE users DROP COLUMN IF EXISTS username;
ALTER TABLE users DROP COLUMN IF EXISTS profile_privacy_status;
