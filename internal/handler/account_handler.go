package handler

import (
	"github.com/dushixiang/pika/internal/service"
	"github.com/go-orz/orz"
	"github.com/labstack/echo/v4"
)

type AccountHandler struct {
	accountService *service.AccountService
}

func NewAccountHandler(accountService *service.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Login 用户登录
func (r AccountHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	loginResp, err := r.accountService.Login(ctx, req.Username, req.Password)
	if err != nil {
		return err
	}

	return orz.Ok(c, loginResp)
}

// Logout 用户登出
func (r AccountHandler) Logout(c echo.Context) error {
	userID := c.Get("userID")
	if userID == nil {
		return orz.NewError(401, "未登录")
	}

	ctx := c.Request().Context()
	if err := r.accountService.Logout(ctx, userID.(string)); err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"message": "登出成功",
	})
}

// ValidateToken 验证 token（供中间件使用）
func (r AccountHandler) ValidateToken(tokenString string) (*service.JWTClaims, error) {
	return r.accountService.ValidateToken(tokenString)
}
