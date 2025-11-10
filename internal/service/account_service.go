package service

import (
	"context"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"github.com/go-errors/errors"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func NewAccountService(logger *zap.Logger, userService *UserService, jwtSecret string, tokenExpireHours int) *AccountService {
	if jwtSecret == "" {
		logger.Fatal("JWT secret cannot be empty")
	}
	if len(jwtSecret) < 32 {
		logger.Warn("JWT secret is too short, should be at least 32 characters for security")
	}
	if tokenExpireHours <= 0 {
		tokenExpireHours = 168 // 默认7天
	}

	service := &AccountService{
		logger:           logger,
		userService:      userService,
		jwtSecret:        jwtSecret,
		tokenExpireHours: tokenExpireHours,
	}
	return service
}

type AccountService struct {
	logger           *zap.Logger
	userService      *UserService
	jwtSecret        string
	tokenExpireHours int
}

// JWTClaims JWT 声明
type JWTClaims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt int64        `json:"expiresAt"`
	User      *models.User `json:"user"`
}

// Login 用户登录
func (s *AccountService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	// 验证用户名密码
	user, err := s.userService.ValidatePassword(ctx, username, password)
	if err != nil {
		return nil, err
	}

	// 生成 JWT token
	expiresAt := time.Now().Add(time.Duration(s.tokenExpireHours) * time.Hour)
	claims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "pika",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.logger.Error("生成token失败", zap.Error(err))
		return nil, errors.New("生成token失败")
	}

	s.logger.Info("用户登录成功", zap.String("userID", user.ID), zap.String("username", user.Username))

	return &LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.UnixMilli(),
		User:      user,
	}, nil
}

// Logout 用户登出
func (s *AccountService) Logout(ctx context.Context, userID string) error {

	s.logger.Info("用户登出成功", zap.String("userID", userID))
	return nil
}

// ValidateToken 验证 JWT token
func (s *AccountService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的token")
}
