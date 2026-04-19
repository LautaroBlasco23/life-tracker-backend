ALTER TABLE users ADD COLUMN username VARCHAR(64);
ALTER TABLE users ADD COLUMN profile_privacy_status VARCHAR(20) NOT NULL DEFAULT 'public';
CREATE UNIQUE INDEX idx_users_username_lower ON users (LOWER(username)) WHERE deleted_at IS NULL;
