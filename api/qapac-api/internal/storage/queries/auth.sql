-- ==========================================================================
-- Auth: Users
-- ==========================================================================

-- name: CreateUser :one
INSERT INTO users (username, password_hash, full_name, phone, role, profile_image_path, active)
VALUES (@username, @password_hash, @full_name, @phone, @role, @profile_image_path, true)
RETURNING id, username, password_hash, full_name, COALESCE(phone, '') AS phone, role,
          COALESCE(profile_image_path, '') AS profile_image_path, active, created_at, updated_at;

-- name: GetUserByUsername :one
SELECT id, username, password_hash, full_name, COALESCE(phone, '') AS phone, role,
       COALESCE(profile_image_path, '') AS profile_image_path, active, created_at, updated_at
FROM users
WHERE username = @username AND active = true;

-- name: GetUserByID :one
SELECT id, username, password_hash, full_name, COALESCE(phone, '') AS phone, role,
       COALESCE(profile_image_path, '') AS profile_image_path, active, created_at, updated_at
FROM users
WHERE id = @id AND active = true;

-- name: ListUsers :many
SELECT id, username, password_hash, full_name, COALESCE(phone, '') AS phone, role,
       COALESCE(profile_image_path, '') AS profile_image_path, active, created_at, updated_at
FROM users
WHERE (sqlc.narg(role)::text IS NULL OR role = sqlc.narg(role)::text)
  AND (sqlc.arg(active_only)::bool = false OR active = true)
ORDER BY created_at DESC;

-- name: UpdateUser :exec
UPDATE users
SET full_name = @full_name, phone = @phone, role = @role,
    profile_image_path = @profile_image_path, active = @active, updated_at = NOW()
WHERE id = @id;

-- name: DeactivateUser :exec
UPDATE users SET active = false, updated_at = NOW() WHERE id = @id;

-- ==========================================================================
-- Auth: Refresh tokens
-- ==========================================================================

-- name: StoreRefreshToken :exec
INSERT INTO refresh_tokens (token_hash, user_id, expires_at)
VALUES (@token_hash, @user_id, @expires_at);

-- name: GetRefreshToken :one
SELECT id, token_hash, user_id, expires_at, revoked, created_at
FROM refresh_tokens
WHERE token_hash = @token_hash;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked = true WHERE token_hash = @token_hash;

-- name: RevokeAllUserTokens :exec
UPDATE refresh_tokens SET revoked = true WHERE user_id = @user_id AND revoked = false;
