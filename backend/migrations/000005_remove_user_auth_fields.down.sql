ALTER TABLE user_snapshots ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE user_snapshots ADD COLUMN auth_provider TEXT NOT NULL DEFAULT 'local';
