package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/model"
	"github.com/msqfX/CertPilot/backend/internal/service"
)

type DashboardHandler struct {
	DB *gorm.DB
}

// Stats 仪表盘统计
func (h *DashboardHandler) Stats(c *gin.Context) {
	uid := c.GetUint("uid")
	var domainCount, certCount, okCount, expiringCount, expiredCount int64
	h.DB.Model(&model.Domain{}).Where("user_id = ?", uid).Count(&domainCount)
	h.DB.Model(&model.Certificate{}).Where("user_id = ?", uid).Count(&certCount)

	now := time.Now()
	in30 := now.AddDate(0, 0, 30)
	h.DB.Model(&model.Certificate{}).Where("user_id = ? AND not_after > ?", uid, in30).Count(&okCount)
	h.DB.Model(&model.Certificate{}).Where("user_id = ? AND not_after > ? AND not_after <= ?", uid, now, in30).Count(&expiringCount)
	h.DB.Model(&model.Certificate{}).Where("user_id = ? AND not_after <= ?", uid, now).Count(&expiredCount)
	c.JSON(http.StatusOK, gin.H{
		"domains": domainCount, "certificates": certCount,
		"ok": okCount, "expiring": expiringCount, "expired": expiredCount,
	})
}

// Upcoming 即将到期的证书日历数据
func (h *DashboardHandler) Upcoming(c *gin.Context) {
	uid := c.GetUint("uid")
	var certs []model.Certificate
	h.DB.Where("user_id = ? AND not_after <= ?", uid, time.Now().AddDate(0, 0, 60)).
		Order("not_after ASC").Limit(30).Find(&certs)
	type item struct {
		ID       uint      `json:"id"`
		Name     string    `json:"name"`
		NotAfter time.Time `json:"not_after"`
		DaysLeft int       `json:"days_left"`
	}
	out := make([]item, 0, len(certs))
	for _, ct := range certs {
		out = append(out, item{ID: ct.ID, Name: ct.CommonName, NotAfter: ct.NotAfter, DaysLeft: int(time.Until(ct.NotAfter).Hours() / 24)})
	}
	c.JSON(http.StatusOK, out)
}

type NotificationHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

type channelReq struct {
	Type   string          `json:"type" binding:"required,oneof=email webhook wecom"`
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config" binding:"required"`
}

// ListChannels 通知渠道列表（配置脱敏，不回传明文）
func (h *NotificationHandler) ListChannels(c *gin.Context) {
	uid := c.GetUint("uid")
	var chans []model.NotificationChannel
	h.DB.Where("user_id = ?", uid).Find(&chans)
	out := make([]gin.H, 0, len(chans))
	for _, ch := range chans {
		out = append(out, gin.H{"id": ch.ID, "type": ch.Type, "name": ch.Name, "enabled": ch.Enabled})
	}
	c.JSON(http.StatusOK, out)
}

func (h *NotificationHandler) CreateChannel(c *gin.Context) {
	uid := c.GetUint("uid")
	var req channelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enc, err := service.Encrypt(h.Cfg.MasterKey, string(req.Config))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ch := model.NotificationChannel{UserID: uid, Type: req.Type, Name: req.Name, ConfigEnc: enc, Enabled: true}
	if err := h.DB.Create(&ch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": ch.ID, "type": ch.Type, "name": ch.Name, "enabled": ch.Enabled})
}

func (h *NotificationHandler) DeleteChannel(c *gin.Context) {
	uid := c.GetUint("uid")
	res := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).Delete(&model.NotificationChannel{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// TestChannel 发送测试通知
func (h *NotificationHandler) TestChannel(c *gin.Context) {
	uid := c.GetUint("uid")
	var ch model.NotificationChannel
	if err := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).First(&ch).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}
	decrypted, err := service.Decrypt(h.Cfg.MasterKey, ch.ConfigEnc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	title := "[CertPilot] 测试通知"
	content := "这是一条来自 CertPilot 的测试通知，如果你看到它，说明渠道配置正确。"
	var cfg service.ChannelConfig
	_ = json.Unmarshal([]byte(decrypted), &cfg)

	var err2 error
	if ch.Type == "email" {
		m := &service.Mailer{
			Host: h.Cfg.SMTPHost, Port: h.Cfg.SMTPPort,
			User: h.Cfg.SMTPUser, Password: h.Cfg.SMTPPassword, From: h.Cfg.SMTPFrom,
		}
		if cfg.To != "" {
			err2 = m.Send(service.SplitCSV(cfg.To), title, content)
		}
	} else {
		err2 = service.SendToChannel(ch.Type, cfg, title, content)
	}
	if err2 != nil {
		h.DB.Create(&model.NotificationLog{UserID: uid, ChannelID: ch.ID, Type: "test", Title: title, Content: content, Status: "fail", Error: err2.Error()})
		c.JSON(http.StatusBadGateway, gin.H{"error": err2.Error()})
		return
	}
	h.DB.Create(&model.NotificationLog{UserID: uid, ChannelID: ch.ID, Type: "test", Title: title, Content: content, Status: "ok"})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Logs 通知发送记录
func (h *NotificationHandler) Logs(c *gin.Context) {
	uid := c.GetUint("uid")
	var logs []model.NotificationLog
	h.DB.Where("user_id = ?", uid).Order("created_at DESC").Limit(100).Find(&logs)
	c.JSON(http.StatusOK, logs)
}

type LogHandler struct{ DB *gorm.DB }

// OperationLogs 操作日志
func (h *LogHandler) List(c *gin.Context) {
	uid := c.GetUint("uid")
	var logs []model.OperationLog
	h.DB.Where("user_id = ?", uid).Order("created_at DESC").Limit(200).Find(&logs)
	c.JSON(http.StatusOK, logs)
}
