ALTER TABLE users
  ADD COLUMN api_key VARCHAR(64) UNIQUE,
  ADD COLUMN api_key_hint VARCHAR(4),
  ADD COLUMN api_key_created_at TIMESTAMPTZ;

CREATE INDEX idx_users_api_key ON users (api_key) WHERE api_key IS NOT NULL;
