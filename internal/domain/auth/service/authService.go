package service

import (
	"errors"
	"fmt"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/auth/dto"
	"life-tracker-backend/internal/domain/auth/model"
	"life-tracker-backend/internal/domain/auth/repository"
	"life-tracker-backend/internal/infrastructure/monitoring"

	userModel "life-tracker-backend/internal/domain/user/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	authRepo repository.AuthRepository
	userRepo repository.UserRepository
	cfg      *config.Config
}

type JWTClaims struct {
	jwt.RegisteredClaims
	Email  string `json:"email"`
	Type   string `json:"type"`
	UserID uint   `json:"userId"`
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		authRepo: repository.NewAuthRepository(db),
		userRepo: repository.NewUserRepository(db),
		cfg:      cfg,
	}
}

func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.TokenResponse, uint, error) {
	exists, err := s.authRepo.EmailExists(req.Email)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		monitoring.AuthAttempts.WithLabelValues("failed_duplicate").Inc()
		return nil, 0, errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, 0, err
	}

	user := &userModel.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, 0, fmt.Errorf("failed to create user: %w", err)
	}

	auth := &model.Auth{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		UserID:       user.ID,
	}

	if err := s.authRepo.Create(auth); err != nil {
		return nil, 0, fmt.Errorf("failed to create auth: %w", err)
	}

	monitoring.UserRegistrations.Inc()
	monitoring.ActiveUsers.Inc()

	tokens, err := s.generateTokens(user.ID, user.Email)
	if err != nil {
		return nil, 0, err
	}

	return tokens, user.ID, nil
}

func (s *AuthService) Login(req *dto.LoginRequest) (*dto.TokenResponse, uint, error) {
	auth, err := s.authRepo.FindByEmail(req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrAuthNotFound) {
			monitoring.AuthAttempts.WithLabelValues("failed_not_found").Inc()
			return nil, 0, errors.New("invalid credentials")
		}
		return nil, 0, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(req.Password)); err != nil {
		monitoring.AuthAttempts.WithLabelValues("failed_password").Inc()
		return nil, 0, errors.New("invalid credentials")
	}

	monitoring.AuthAttempts.WithLabelValues("success").Inc()

	tokens, err := s.generateTokens(auth.UserID, auth.Email)
	if err != nil {
		return nil, 0, err
	}

	return tokens, auth.UserID, nil
}

func (s *AuthService) RefreshToken(refreshToken string) (*dto.TokenResponse, error) {
	claims, err := s.validateToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type")
	}

	return s.generateTokens(claims.UserID, claims.Email)
}

func (s *AuthService) generateTokens(userID uint, email string) (*dto.TokenResponse, error) {
	accessExpiry, err := time.ParseDuration(s.cfg.JWTExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT expiry duration: %w", err)
	}
	refreshExpiry, err := time.ParseDuration(s.cfg.JWTRefreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT refresh expiry duration: %w", err)
	}

	now := time.Now()

	accessClaims := &JWTClaims{
		UserID: userID,
		Email:  email,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	refreshClaims := &JWTClaims{
		UserID: userID,
		Email:  email,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &dto.TokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(accessExpiry.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(_ *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
