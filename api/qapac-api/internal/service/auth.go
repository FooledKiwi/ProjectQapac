package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Sentinel errors for the auth service.
var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrTokenExpired       = errors.New("auth: token expired")
	ErrTokenRevoked       = errors.New("auth: token revoked")
	ErrJWTSecretMissing   = errors.New("auth: JWT_SECRET not configured")
)

// TokenPair holds an access token and refresh token pair.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthClaims are the JWT claims embedded in access tokens.
type AuthClaims struct {
	jwt.RegisteredClaims
	UserID   int32  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// AuthService handles authentication logic: login, token refresh, and logout.
type AuthService struct {
	usersRepo  storage.UsersRepository
	tokensRepo storage.RefreshTokensRepository
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewAuthService creates an AuthService with the given dependencies.
func NewAuthService(
	usersRepo storage.UsersRepository,
	tokensRepo storage.RefreshTokensRepository,
	jwtSecret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		usersRepo:  usersRepo,
		tokensRepo: tokensRepo,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Login authenticates a user by username and password, returning a token pair.
func (s *AuthService) Login(ctx context.Context, username, password string) (*TokenPair, *storage.User, error) {
	if len(s.jwtSecret) == 0 {
		return nil, nil, ErrJWTSecretMissing
	}

	user, err := s.usersRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: lookup user: %w", err)
	}
	if user == nil {
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	pair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return pair, user, nil
}

// Refresh validates a refresh token and issues a new token pair.
// The old refresh token is revoked (rotation).
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	if len(s.jwtSecret) == 0 {
		return nil, ErrJWTSecretMissing
	}

	tokenHash := hashToken(rawRefreshToken)

	stored, err := s.tokensRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("auth: lookup refresh token: %w", err)
	}
	if stored == nil {
		return nil, ErrInvalidCredentials
	}
	if stored.Revoked {
		return nil, ErrTokenRevoked
	}
	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Revoke old token (rotation).
	if err := s.tokensRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("auth: revoke old token: %w", err)
	}

	user, err := s.usersRepo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth: lookup user for refresh: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	pair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	return pair, nil
}

// Logout revokes a specific refresh token.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	tokenHash := hashToken(rawRefreshToken)
	if err := s.tokensRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return fmt.Errorf("auth: revoke token on logout: %w", err)
	}
	return nil
}

// ValidateAccessToken parses and validates an access token, returning the claims.
func (s *AuthService) ValidateAccessToken(tokenString string) (*AuthClaims, error) {
	if len(s.jwtSecret) == 0 {
		return nil, ErrJWTSecretMissing
	}

	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: parse access token: %w", err)
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidCredentials
	}

	return claims, nil
}

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// generateTokenPair creates an access + refresh token pair for the given user.
func (s *AuthService) generateTokenPair(ctx context.Context, user *storage.User) (*TokenPair, error) {
	now := time.Now()

	// Access token (JWT).
	claims := AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			Issuer:    "qapac-api",
		},
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("auth: sign access token: %w", err)
	}

	// Refresh token (opaque random string, stored as hash).
	rawRefresh, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("auth: generate refresh token: %w", err)
	}

	refreshHash := hashToken(rawRefresh)
	expiresAt := now.Add(s.refreshTTL)

	if err := s.tokensRepo.StoreRefreshToken(ctx, refreshHash, user.ID, expiresAt); err != nil {
		return nil, fmt.Errorf("auth: store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

// generateRandomToken produces a hex-encoded random string of n bytes.
func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex digest of a token string.
// We store hashes instead of raw tokens for security.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
