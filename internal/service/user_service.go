// Package service 提供业务逻辑层实现。
// 服务层负责协调领域对象和仓储，实现具体的业务用例。
package service

import (
	"errors"
	"fmt"
	"github.com/danta7/go_mall/internal/domain"
	"github.com/danta7/go_mall/internal/repo"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
)

// UserService 定义用户服务接口
type UserService interface {
	Register(req *domain.RegisterRequest) (*domain.User, error)
	Login(req *domain.LoginRequest) (*domain.User, error)
	GetUserByID(id int64) (*domain.User, error)
	GetUserByUsername(username string) (*domain.User, error)
}

type userService struct {
	userRepo repo.UserRepository
	logger   *zap.Logger
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repo.UserRepository, logger *zap.Logger) UserService {
	return &userService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Register 用户注册
// 业务规则：
// 1. 用户名和邮箱不能重复
// 2. 密码需要进行bcrypt哈希
// 3. 新用户默认为普通用户角色
func (s *userService) Register(req *domain.RegisterRequest) (*domain.User, error) {
	// 验证用户名是否存在
	existingUser, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		s.logger.Error("failed to check username", zap.Error(err))
		return nil, fmt.Errorf("check username: %w", err)
	}

	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 验证邮箱是否存在
	existingUser, err = s.userRepo.GetByEmail(req.Email)
	if err != nil {
		s.logger.Error("failed to check email", zap.Error(err))
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 哈希密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Username:     strings.TrimSpace(req.Username),
		Email:        strings.TrimSpace(strings.ToLower(req.Email)),
		PasswordHash: string(passwordHash),
		Role:         domain.UserRoleUser,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, fmt.Errorf("create user: %w", err)
	}

	s.logger.Info("user registered successfully", zap.Int64("user_id", user.ID), zap.String("username", user.Username))
	return user, nil
}

// Login 用户登录
// 业务规则：
// 1. 支持用户名或邮箱登录
// 2. 验证密码正确性
// 3. 检查用户是否处于活跃状态
func (s *userService) Login(req *domain.LoginRequest) (*domain.User, error) {
	// 尝试通过用户名查找用户
	user, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		s.logger.Error("failed to get user by username", zap.Error(err))
		return nil, fmt.Errorf("get user: %w", err)
	}

	// 如果用户名找不到，尝试用邮箱查找
	if user == nil {
		user, err = s.userRepo.GetByEmail(req.Username)
		if err != nil {
			s.logger.Error("failed to get user by email", zap.Error(err))
			return nil, fmt.Errorf("get user: %w", err)
		}
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// 验证密码
	// bcrypt.CompareHashAndPassword 会自动处理盐值和哈希比较
	// 这个函数具有时间恒定特性，可以防止时序攻击
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return nil, ErrInvalidCredentials
		}
		s.logger.Error("failed to compare password", zap.Error(err))
		return nil, fmt.Errorf("compare password: %w", err)
	}

	s.logger.Info("user logged in successfully",
		zap.Int64("user_id", user.ID),
		zap.String("username", user.Username),
	)

	return user, nil
}

// GetUserByID 根据ID获取用户
func (s *userService) GetUserByID(id int64) (*domain.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		s.logger.Error("failed to get user by id", zap.Int64("id", id), zap.Error(err))
		return nil, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *userService) GetUserByUsername(username string) (*domain.User, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		s.logger.Error("failed to get user by username", zap.String("username", username), zap.Error(err))
		return nil, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}
