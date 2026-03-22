package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/doomslock/backend/config"
	"github.com/doomslock/backend/internal/auth/repository"
	"github.com/doomslock/backend/pkg/middleware"
)

var (
	ErrEmailTaken    = errors.New("email already registered")
	ErrUsernameTaken = errors.New("username already taken")
	ErrInvalidCreds  = errors.New("invalid email or password")
	ErrTokenInvalid  = errors.New("invalid or expired refresh token")
)

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Timezone string `json:"timezone"`
	FCMToken string `json:"fcm_token"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
	FCMToken string `json:"fcm_token"`
}

type AuthResponse struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	Logout(ctx context.Context, userID, refreshToken string) error
}

type service struct {
	repo repository.Repository
	rdb  *goredis.Client
	cfg  config.JWTConfig
}

func New(repo repository.Repository, rdb *goredis.Client, cfg config.JWTConfig) Service {
	return &service{repo: repo, rdb: rdb, cfg: cfg}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	existing, _ := s.repo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	tz := req.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	user := &repository.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		FCMToken:     req.FCMToken,
		Timezone:     tz,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.issueTokens(ctx, user)
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCreds
	}

	if req.FCMToken != "" && req.FCMToken != user.FCMToken {
		_ = s.repo.UpdateFCMToken(ctx, user.ID, req.FCMToken)
	}

	return s.issueTokens(ctx, user)
}

func (s *service) RefreshToken(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	key := "refresh:" + refreshToken
	userID, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, ErrTokenInvalid
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	accessToken, err := s.newAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	ttl := time.Duration(s.cfg.AccessTokenTTL) * time.Minute
	return &RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(ttl.Seconds()),
	}, nil
}

func (s *service) Logout(ctx context.Context, userID, refreshToken string) error {
	return s.rdb.Del(ctx, "refresh:"+refreshToken).Err()
}

// --- internal ---

func (s *service) issueTokens(ctx context.Context, user *repository.User) (*AuthResponse, error) {
	accessToken, err := s.newAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("access token: %w", err)
	}

	refreshToken := uuid.New().String()
	refreshTTL := time.Duration(s.cfg.RefreshTokenTTL) * 24 * time.Hour
	if err := s.rdb.Set(ctx, "refresh:"+refreshToken, user.ID, refreshTTL).Err(); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	ttl := time.Duration(s.cfg.AccessTokenTTL) * time.Minute
	return &AuthResponse{
		UserID:       user.ID,
		Username:     user.Username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(ttl.Seconds()),
	}, nil
}

func (s *service) newAccessToken(user *repository.User) (string, error) {
	ttl := time.Duration(s.cfg.AccessTokenTTL) * time.Minute
	claims := &middleware.JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}
