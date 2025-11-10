package service

import (
	"context"
	"errors"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"github.com/dushixiang/pika/internal/repo"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	logger     *zap.Logger
	userRepo   *repo.UserRepo
	Repository *repo.UserRepo
}

func NewUserService(logger *zap.Logger, userRepo *repo.UserRepo) *UserService {
	return &UserService{
		logger:     logger,
		userRepo:   userRepo,
		Repository: userRepo,
	}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*models.User, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, errors.New("用户名已存在")
	}
	// 如果不是"记录不存在"的错误，说明是数据库错误
	if err != nil && err.Error() != "record not found" {
		s.logger.Error("查询用户名失败", zap.Error(err), zap.String("username", req.Username))
		return nil, errors.New("系统错误，请稍后重试")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()
	user := &models.User{
		ID:        uuid.NewString(),
		Username:  req.Username,
		Password:  string(hashedPassword),
		Nickname:  req.Nickname,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	s.logger.Info("user created successfully", zap.String("userID", user.ID), zap.String("username", user.Username))
	return user, nil
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(ctx context.Context, userID string, req *UpdateUserRequest) (*models.User, error) {
	user, err := s.userRepo.FindById(ctx, userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	// 更新字段
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}

	if err := s.userRepo.UpdateById(ctx, &user); err != nil {
		return nil, err
	}

	s.logger.Info("user updated successfully", zap.String("userID", userID))
	return &user, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	if err := s.userRepo.DeleteById(ctx, userID); err != nil {
		return err
	}

	s.logger.Info("user deleted successfully", zap.String("userID", userID))
	return nil
}

// GetUser 获取用户信息
func (s *UserService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.FindById(ctx, userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.userRepo.FindByUsername(ctx, username)
}

// ListUsers 列出用户
func (s *UserService) ListUsers(ctx context.Context, page, pageSize int) ([]models.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.userRepo.ListUsers(ctx, offset, pageSize)
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(ctx context.Context, userID string, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindById(ctx, userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return err
	}

	s.logger.Info("password changed successfully", zap.String("userID", userID))
	return nil
}

// ResetPassword 重置密码（管理员操作）
func (s *UserService) ResetPassword(ctx context.Context, userID, newPassword string) error {
	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return err
	}

	s.logger.Info("password reset successfully", zap.String("userID", userID))
	return nil
}

// UpdateUserStatus 更新用户状态
func (s *UserService) UpdateUserStatus(ctx context.Context, userID string, status int) error {
	if err := s.userRepo.UpdateStatus(ctx, userID, status); err != nil {
		return err
	}

	s.logger.Info("user status updated successfully", zap.String("userID", userID), zap.Int("status", status))
	return nil
}

// ValidatePassword 验证密码
func (s *UserService) ValidatePassword(ctx context.Context, username, password string) (*models.User, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return user, nil
}

// 数据结构定义
type CreateUserRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
	Nickname string `json:"nickname"`
}

type UpdateUserRequest struct {
	Nickname string `json:"nickname"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=6"`
}
