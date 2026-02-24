package storage

import (
	"context"
	"time"
)

// User represents a driver or admin account.
type User struct {
	ID               int32
	Username         string
	PasswordHash     string
	FullName         string
	Phone            string
	Role             string // "driver" or "admin"
	ProfileImagePath string
	Active           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// RefreshToken represents a stored JWT refresh token.
type RefreshToken struct {
	ID        int32
	TokenHash string
	UserID    int32
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

// UsersRepository defines operations on the users table.
type UsersRepository interface {
	// CreateUser inserts a new user and returns it with the generated ID.
	CreateUser(ctx context.Context, u *User) (*User, error)

	// GetUserByUsername returns a user by username, or (nil, nil) if not found.
	GetUserByUsername(ctx context.Context, username string) (*User, error)

	// GetUserByID returns a user by ID, or (nil, nil) if not found.
	GetUserByID(ctx context.Context, id int32) (*User, error)

	// ListUsers returns users filtered by optional role and active status.
	// Pass empty role to list all roles.
	ListUsers(ctx context.Context, role string, activeOnly bool) ([]User, error)

	// UpdateUser updates mutable fields on a user.
	UpdateUser(ctx context.Context, u *User) error

	// DeactivateUser performs a soft-delete by setting active = false.
	DeactivateUser(ctx context.Context, id int32) error
}

// RefreshTokensRepository defines operations on the refresh_tokens table.
type RefreshTokensRepository interface {
	// StoreRefreshToken persists a hashed refresh token.
	StoreRefreshToken(ctx context.Context, tokenHash string, userID int32, expiresAt time.Time) error

	// GetRefreshToken returns a refresh token by hash, or (nil, nil) if not found.
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)

	// RevokeRefreshToken marks a refresh token as revoked.
	RevokeRefreshToken(ctx context.Context, tokenHash string) error

	// RevokeAllUserTokens revokes all refresh tokens for a user.
	RevokeAllUserTokens(ctx context.Context, userID int32) error
}
