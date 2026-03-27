ALTER TABLE user_snapshots DROP COLUMN IF EXISTS password_hash;
ALTER TABLE user_snapshots DROP COLUMN IF EXISTS auth_provider;
