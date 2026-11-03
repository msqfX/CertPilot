package service

import (
	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/model"
	"gorm.io/gorm"
)

// Notifier 统一通知出口：向用户所有启用的渠道发送通知并落日志。
// Scheduler（到期提醒）与 ACME 引擎（签发/续期结果）共用。
type Notifier struct {
	DB     *gorm.DB
	Cfg    *config.Config
	Mailer *Mailer
}

func (n *Notifier) Notify(userID uint, certID *uint, typ, title, content string) {
	var channels []model.NotificationChannel
	n.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&channels)
	for _, ch := range channels {
		status, errMsg := "ok", ""
		cfg := n.decodedConfig(ch.ConfigEnc)
		if ch.Type == "email" {
			m := *n.Mailer
			if cfg.To != "" {
				if err := m.Send(SplitCSV(cfg.To), title, content); err != nil {
					status, errMsg = "fail", err.Error()
				}
			}
		} else {
			if err := SendToChannel(ch.Type, cfg, title, content); err != nil {
				status, errMsg = "fail", err.Error()
			}
		}
		n.DB.Create(&model.NotificationLog{
			UserID: userID, ChannelID: ch.ID, CertID: certID,
			Type: typ, Title: title, Content: content, Status: status, Error: errMsg,
		})
	}
}

func (n *Notifier) decodedConfig(enc string) ChannelConfig {
	var cfg ChannelConfig
	if plain, err := Decrypt(n.Cfg.MasterKey, enc); err == nil {
		_ = jsonUnmarshal(plain, &cfg)
	}
	return cfg
}
