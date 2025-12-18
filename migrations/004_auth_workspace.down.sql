-- Remove workspace_id from existing tables
ALTER TABLE workflows DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE credentials DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE schedules DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE webhooks DROP COLUMN IF EXISTS workspace_id;

-- Remove auth columns from users
ALTER TABLE users DROP COLUMN IF EXISTS failed_attempts;
ALTER TABLE users DROP COLUMN IF EXISTS locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;

-- Drop audit logs
DROP TABLE IF EXISTS audit_logs;

-- Drop workspace invitations
DROP TABLE IF EXISTS workspace_invitations;

-- Drop workspace members
DROP TABLE IF EXISTS workspace_members;

-- Drop workspaces
DROP TABLE IF EXISTS workspaces;

-- Drop login attempts
DROP TABLE IF EXISTS login_attempts;

-- Drop OAuth connections
DROP TABLE IF EXISTS oauth_connections;

-- Drop API keys
DROP TABLE IF EXISTS api_keys;

-- Drop email verification tokens
DROP TABLE IF EXISTS email_verification_tokens;

-- Drop password reset tokens
DROP TABLE IF EXISTS password_reset_tokens;

-- Drop refresh tokens
DROP TABLE IF EXISTS refresh_tokens;
