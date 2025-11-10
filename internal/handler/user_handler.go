package handler

import (
	"github.com/dushixiang/pika/internal/service"
	"github.com/go-orz/orz"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" validate:"required,min=6"`
}

// UpdateUserStatusRequest 更新用户状态请求
type UpdateUserStatusRequest struct {
	Status int `json:"status" validate:"required,oneof=0 1"`
}

// Paging 用户分页查询
func (r UserHandler) Paging(c echo.Context) error {
	username := c.QueryParam("username")

	pr := orz.GetPageRequest(c, "created_at", "username")

	builder := orz.NewPageBuilder(r.userService.Repository).
		PageRequest(pr).
		Contains("username", username)

	ctx := c.Request().Context()
	page, err := builder.Execute(ctx)
	if err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"items": page.Items,
		"total": page.Total,
	})
}

// Create 创建用户
func (r UserHandler) Create(c echo.Context) error {
	var req service.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	user, err := r.userService.CreateUser(ctx, &req)
	if err != nil {
		return err
	}

	return orz.Ok(c, user)
}

// Get 获取用户详情
func (r UserHandler) Get(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	user, err := r.userService.GetUser(ctx, id)
	if err != nil {
		return err
	}

	return orz.Ok(c, user)
}

// Update 更新用户
func (r UserHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req service.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	user, err := r.userService.UpdateUser(ctx, id, &req)
	if err != nil {
		return err
	}

	return orz.Ok(c, user)
}

// Delete 删除用户
func (r UserHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	if err := r.userService.DeleteUser(ctx, id); err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"message": "用户删除成功",
	})
}

// ChangePassword 修改密码
func (r UserHandler) ChangePassword(c echo.Context) error {
	id := c.Param("id")

	var req service.ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	if err := r.userService.ChangePassword(ctx, id, req.OldPassword, req.NewPassword); err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"message": "密码修改成功",
	})
}

// ResetPassword 重置密码（管理员操作）
func (r UserHandler) ResetPassword(c echo.Context) error {
	id := c.Param("id")

	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	if err := r.userService.ResetPassword(ctx, id, req.NewPassword); err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"message": "密码重置成功",
	})
}

// UpdateStatus 更新用户状态
func (r UserHandler) UpdateStatus(c echo.Context) error {
	id := c.Param("id")

	var req UpdateUserStatusRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	if err := r.userService.UpdateUserStatus(ctx, id, req.Status); err != nil {
		return err
	}

	return orz.Ok(c, orz.Map{
		"message": "用户状态更新成功",
	})
}
