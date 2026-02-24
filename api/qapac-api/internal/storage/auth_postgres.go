package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgUsersRepository is the pgx-backed implementation of UsersRepository.
type pgUsersRepository struct {
	pool *pgxpool.Pool
}

// NewUsersRepository creates a UsersRepository backed by the given connection pool.
func NewUsersRepository(pool *pgxpool.Pool) UsersRepository {
	return &pgUsersRepository{pool: pool}
}

func (r *pgUsersRepository) CreateUser(ctx context.Context, u *User) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var id int32
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, full_name, phone, role, profile_image_path, active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		u.Username, u.PasswordHash, u.FullName, u.Phone, u.Role, u.ProfileImagePath, true,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("storage: CreateUser: %w", err)
	}

	u.ID = id
	u.Active = true
	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt
	return u, nil
}

func (r *pgUsersRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	u := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, full_name, COALESCE(phone, ''), role,
		        COALESCE(profile_image_path, ''), active, created_at, updated_at
		 FROM users
		 WHERE username = $1 AND active = true`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role,
		&u.ProfileImagePath, &u.Active, &u.CreatedAt, &u.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetUserByUsername: %w", err)
	}
	return u, nil
}

func (r *pgUsersRepository) GetUserByID(ctx context.Context, id int32) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	u := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, full_name, COALESCE(phone, ''), role,
		        COALESCE(profile_image_path, ''), active, created_at, updated_at
		 FROM users
		 WHERE id = $1 AND active = true`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role,
		&u.ProfileImagePath, &u.Active, &u.CreatedAt, &u.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetUserByID: %w", err)
	}
	return u, nil
}

// pgRefreshTokensRepository is the pgx-backed implementation of RefreshTokensRepository.
type pgRefreshTokensRepository struct {
	pool *pgxpool.Pool
}

// NewRefreshTokensRepository creates a RefreshTokensRepository backed by the given pool.
func NewRefreshTokensRepository(pool *pgxpool.Pool) RefreshTokensRepository {
	return &pgRefreshTokensRepository{pool: pool}
}

func (r *pgRefreshTokensRepository) StoreRefreshToken(ctx context.Context, tokenHash string, userID int32, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (token_hash, user_id, expires_at)
		 VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("storage: StoreRefreshToken: %w", err)
	}
	return nil
}

func (r *pgRefreshTokensRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rt := &RefreshToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, token_hash, user_id, expires_at, revoked, created_at
		 FROM refresh_tokens
		 WHERE token_hash = $1`,
		tokenHash,
	).Scan(&rt.ID, &rt.TokenHash, &rt.UserID, &rt.ExpiresAt, &rt.Revoked, &rt.CreatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetRefreshToken: %w", err)
	}
	return rt, nil
}

func (r *pgRefreshTokensRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("storage: RevokeRefreshToken: %w", err)
	}
	return nil
}

func (r *pgRefreshTokensRepository) RevokeAllUserTokens(ctx context.Context, userID int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = true WHERE user_id = $1 AND revoked = false`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("storage: RevokeAllUserTokens: %w", err)
	}
	return nil
}
