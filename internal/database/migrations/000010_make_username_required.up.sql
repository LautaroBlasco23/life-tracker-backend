-- Make username NOT NULL since it's now required during registration
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
