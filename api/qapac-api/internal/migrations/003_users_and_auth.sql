-- Migration: 003_users_and_auth
-- Adds user accounts (drivers, admins) and refresh token tracking for JWT auth.
-- Normal transit users are anonymous and do not need accounts.

-- =====================
-- Users (drivers and admins only)
-- =====================

CREATE TABLE IF NOT EXISTS users (
  id                 SERIAL PRIMARY KEY,
  username           VARCHAR(100)  NOT NULL UNIQUE,
  password_hash      VARCHAR(255)  NOT NULL,
  full_name          VARCHAR(255)  NOT NULL,
  phone              VARCHAR(20),
  role               VARCHAR(20)   NOT NULL CHECK (role IN ('driver', 'admin')),
  profile_image_path VARCHAR(500),
  active             BOOLEAN       DEFAULT true,
  created_at         TIMESTAMP     DEFAULT NOW(),
  updated_at         TIMESTAMP     DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_role   ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);

-- =====================
-- Refresh tokens for JWT session management
-- =====================

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id         SERIAL PRIMARY KEY,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  user_id    INT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMP    NOT NULL,
  revoked    BOOLEAN      DEFAULT false,
  created_at TIMESTAMP    DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
