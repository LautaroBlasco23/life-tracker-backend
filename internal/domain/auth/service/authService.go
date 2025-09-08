package service

import (
	"errors"
	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/auth/dto"
	"life-tracker-backend/internal/domain/auth/model"
	"time"

	userModel "life-tracker-backend/internal/domain/user/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

type JWTClaims struct {
	UserID uint   `json:"userId"`
	Email  string `json:"email"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		db:  db,
		cfg: cfg,
	}
}

func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.TokenResponse, error) {
	// Check if user already exists
	var existingAuth model.Auth
	if err := s.db.Where("email = ?", req.Email).First(&existingAuth).Error; err == nil {
		return nil, errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Start transaction
	tx := s.db.Begin()

	// Create user
	user := &userModel.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	}

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Create auth record
	auth := &model.Auth{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		UserID:       user.ID,
	}

	if err := tx.Create(auth).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	// Generate tokens
	return s.generateTokens(user.ID, user.Email)
}

func (s *AuthService) Login(req *dto.LoginRequest) (*dto.TokenResponse, error) {
	var auth model.Auth
	if err := s.db.Where("email = ?", req.Email).First(&auth).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return s.generateTokens(auth.UserID, auth.Email)
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
	// Parse expiry duration
	accessExpiry, _ := time.ParseDuration(s.cfg.JWTExpiry)
	refreshExpiry, _ := time.ParseDuration(s.cfg.JWTRefreshExpiry)

	now := time.Now()

	// Generate access token
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

	// Generate refresh token
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
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
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
