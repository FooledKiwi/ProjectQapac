package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/generated/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgUsersRepository is the pgx-backed implementation of UsersRepository.
type pgUsersRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// NewUsersRepository creates a UsersRepository backed by the given connection pool.
func NewUsersRepository(pool *pgxpool.Pool) UsersRepository {
	return &pgUsersRepository{q: db.New(pool), pool: pool}
}

func (r *pgUsersRepository) CreateUser(ctx context.Context, u *User) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Username:         u.Username,
		PasswordHash:     u.PasswordHash,
		FullName:         u.FullName,
		Phone:            pgtext(u.Phone),
		Role:             u.Role,
		ProfileImagePath: pgtext(u.ProfileImagePath),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: CreateUser: %w", err)
	}

	u.ID = row.ID
	u.Active = row.Active.Bool
	u.Phone = row.Phone
	u.ProfileImagePath = row.ProfileImagePath
	u.CreatedAt = row.CreatedAt.Time
	u.UpdatedAt = row.UpdatedAt.Time
	return u, nil
}

func (r *pgUsersRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetUserByUsername: %w", err)
	}

	return userRowToUser(row.ID, row.Username, row.PasswordHash, row.FullName,
		row.Phone, row.Role, row.ProfileImagePath, row.Active, row.CreatedAt, row.UpdatedAt), nil
}

func (r *pgUsersRepository) GetUserByID(ctx context.Context, id int32) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetUserByID: %w", err)
	}

	return userRowToUser(row.ID, row.Username, row.PasswordHash, row.FullName,
		row.Phone, row.Role, row.ProfileImagePath, row.Active, row.CreatedAt, row.UpdatedAt), nil
}

func (r *pgUsersRepository) ListUsers(ctx context.Context, role string, activeOnly bool) ([]User, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var roleParam pgtype.Text
	if role != "" {
		roleParam = pgtype.Text{String: role, Valid: true}
	}

	rows, err := r.q.ListUsers(ctx, db.ListUsersParams{
		Role:       roleParam,
		ActiveOnly: activeOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: ListUsers: %w", err)
	}

	users := make([]User, 0, len(rows))
	for _, row := range rows {
		users = append(users, *userRowToUser(row.ID, row.Username, row.PasswordHash, row.FullName,
			row.Phone, row.Role, row.ProfileImagePath, row.Active, row.CreatedAt, row.UpdatedAt))
	}
	return users, nil
}

func (r *pgUsersRepository) UpdateUser(ctx context.Context, u *User) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.UpdateUser(ctx, db.UpdateUserParams{
		FullName:         u.FullName,
		Phone:            pgtext(u.Phone),
		Role:             u.Role,
		ProfileImagePath: pgtext(u.ProfileImagePath),
		Active:           pgbool(u.Active),
		ID:               u.ID,
	})
	if err != nil {
		return fmt.Errorf("storage: UpdateUser: %w", err)
	}
	return nil
}

func (r *pgUsersRepository) DeactivateUser(ctx context.Context, id int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.DeactivateUser(ctx, id)
	if err != nil {
		return fmt.Errorf("storage: DeactivateUser: %w", err)
	}
	return nil
}

// pgRefreshTokensRepository is the pgx-backed implementation of RefreshTokensRepository.
type pgRefreshTokensRepository struct {
	q *db.Queries
}

// NewRefreshTokensRepository creates a RefreshTokensRepository backed by the given pool.
func NewRefreshTokensRepository(pool *pgxpool.Pool) RefreshTokensRepository {
	return &pgRefreshTokensRepository{q: db.New(pool)}
}

func (r *pgRefreshTokensRepository) StoreRefreshToken(ctx context.Context, tokenHash string, userID int32, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.StoreRefreshToken(ctx, db.StoreRefreshTokenParams{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("storage: StoreRefreshToken: %w", err)
	}
	return nil
}

func (r *pgRefreshTokensRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetRefreshToken(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetRefreshToken: %w", err)
	}

	return &RefreshToken{
		ID:        row.ID,
		TokenHash: row.TokenHash,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt.Time,
		Revoked:   row.Revoked.Bool,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *pgRefreshTokensRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.RevokeRefreshToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("storage: RevokeRefreshToken: %w", err)
	}
	return nil
}

func (r *pgRefreshTokensRepository) RevokeAllUserTokens(ctx context.Context, userID int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("storage: RevokeAllUserTokens: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// userRowToUser maps a sqlc-generated user row into the domain User struct.
func userRowToUser(
	id int32, username, passwordHash, fullName, phone, role, profileImagePath string,
	active pgtype.Bool, createdAt, updatedAt pgtype.Timestamp,
) *User {
	return &User{
		ID:               id,
		Username:         username,
		PasswordHash:     passwordHash,
		FullName:         fullName,
		Phone:            phone,
		Role:             role,
		ProfileImagePath: profileImagePath,
		Active:           active.Bool,
		CreatedAt:        createdAt.Time,
		UpdatedAt:        updatedAt.Time,
	}
}

// pgtext builds a pgtype.Text from a Go string.
// Empty strings are stored as NULL (Valid=false).
func pgtext(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// pgbool builds a pgtype.Bool from a Go bool.
func pgbool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}
