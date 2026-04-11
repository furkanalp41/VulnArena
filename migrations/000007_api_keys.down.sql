DROP INDEX IF EXISTS idx_users_api_key;

ALTER TABLE users
  DROP COLUMN IF EXISTS api_key_created_at,
  DROP COLUMN IF EXISTS api_key_hint,
  DROP COLUMN IF EXISTS api_key;
