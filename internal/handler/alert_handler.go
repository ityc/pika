package handler

import (
	"net/http"
	"strconv"

	"github.com/dushixiang/pika/internal/models"
	"github.com/dushixiang/pika/internal/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type AlertHandler struct {
	logger       *zap.Logger
	alertService *service.AlertService
}

func NewAlertHandler(logger *zap.Logger, alertService *service.AlertService) *AlertHandler {
	return &AlertHandler{
		logger:       logger,
		alertService: alertService,
	}
}

// CreateAlertConfig 创建告警配置
func (h *AlertHandler) CreateAlertConfig(c echo.Context) error {
	var config models.AlertConfig
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "无效的请求参数",
		})
	}

	if err := h.alertService.CreateAlertConfig(c.Request().Context(), &config); err != nil {
		h.logger.Error("创建告警配置失败", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "创建告警配置失败",
		})
	}

	return c.JSON(http.StatusOK, config)
}

// UpdateAlertConfig 更新告警配置
func (h *AlertHandler) UpdateAlertConfig(c echo.Context) error {
	id := c.Param("id")

	var config models.AlertConfig
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "无效的请求参数",
		})
	}

	config.ID = id

	if err := h.alertService.UpdateAlertConfig(c.Request().Context(), &config); err != nil {
		h.logger.Error("更新告警配置失败", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "更新告警配置失败",
		})
	}

	return c.JSON(http.StatusOK, config)
}

// DeleteAlertConfig 删除告警配置
func (h *AlertHandler) DeleteAlertConfig(c echo.Context) error {
	id := c.Param("id")

	if err := h.alertService.DeleteAlertConfig(c.Request().Context(), id); err != nil {
		h.logger.Error("删除告警配置失败", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "删除告警配置失败",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "删除成功",
	})
}

// GetAlertConfig 获取告警配置
func (h *AlertHandler) GetAlertConfig(c echo.Context) error {
	id := c.Param("id")

	config, err := h.alertService.GetAlertConfig(c.Request().Context(), id)
	if err != nil {
		h.logger.Error("获取告警配置失败", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "获取告警配置失败",
		})
	}

	return c.JSON(http.StatusOK, config)
}

// ListAlertConfigsByAgent 列出探针的告警配置
func (h *AlertHandler) ListAlertConfigsByAgent(c echo.Context) error {
	agentID := c.Param("agentId")

	configs, err := h.alertService.ListAlertConfigsByAgent(c.Request().Context(), agentID)
	if err != nil {
		h.logger.Error("获取告警配置列表失败", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "获取告警配置列表失败",
		})
	}

	return c.JSON(http.StatusOK, configs)
}

// ListAlertRecords 列出告警记录
func (h *AlertHandler) ListAlertRecords(c echo.Context) error {
	agentID := c.QueryParam("agentId")

	// 解析分页参数
	limit := 20
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if o := c.QueryParam("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	records, total, err := h.alertService.ListAlertRecords(c.Request().Context(), agentID, limit, offset)
	if err != nil {
		h.logger.Error("获取告警记录失败", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "获取告警记录失败",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"records": records,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// TestNotification 测试告警通知
func (h *AlertHandler) TestNotification(c echo.Context) error {
	id := c.Param("id")

	if err := h.alertService.TestNotification(c.Request().Context(), id); err != nil {
		h.logger.Error("测试告警通知失败", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "测试告警通知失败: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "测试通知已发送",
	})
}
