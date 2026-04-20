-- Revert: make username nullable again
ALTER TABLE users ALTER COLUMN username DROP NOT NULL;
