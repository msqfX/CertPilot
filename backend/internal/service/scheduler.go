package service

import (
	"fmt"
	"log"
	"time"

	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/model"
	"gorm.io/gorm"
)

// Scheduler 到期巡检：每天 09:00 扫描所有域名/证书，按阈值发送提醒。
type Scheduler struct {
	DB     *gorm.DB
	Cfg    *config.Config
	Mailer *Mailer
}

func (s *Scheduler) Start() {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(next.Sub(now))
			s.Run()
		}
	}()
	log.Println("[scheduler] expiry check scheduled daily at 09:00")
}

// Run 执行一轮巡检：重新检测所有域名的 HTTPS 证书，并对命中阈值的通知发提醒。
func (s *Scheduler) Run() {
	var domains []model.Domain
	s.DB.Find(&domains)
	for _, d := range domains {
		info, err := CheckHTTPS(d.Name, 443)
		if err != nil {
			s.DB.Model(&d).Updates(map[string]any{"last_check_at": time.Now(), "last_error": err.Error()})
			continue
		}
		s.DB.Model(&d).Updates(map[string]any{"last_check_at": time.Now(), "last_error": ""})

		cert := s.upsertCert(d.UserID, *info, d.ID)
		if cert == nil {
			continue
		}
		for _, days := range s.Cfg.ExpiryRemindDays {
			left := int(time.Until(cert.NotAfter).Hours() / 24)
			if left == days {
				s.notifyExpiry(d.UserID, cert, days)
				break
			}
		}
	}
}

func (s *Scheduler) upsertCert(userID uint, info CertInfo, domainID uint) *model.Certificate {
	var cert model.Certificate
	err := s.DB.Where("fingerprint = ?", info.Fingerprint).First(&cert).Error
	if err != nil {
		cert = model.Certificate{
			UserID:      userID,
			Name:        info.CommonName,
			CommonName:  info.CommonName,
			Issuer:      info.Issuer,
			NotBefore:   info.NotBefore,
			NotAfter:    info.NotAfter,
			Fingerprint: info.Fingerprint,
			Source:      "monitor",
		}
		if err := s.DB.Create(&cert).Error; err != nil {
			log.Println("[scheduler] create cert:", err)
			return nil
		}
	}
	// 关联域名
	var count int64
	s.DB.Model(&model.CertificateDomain{}).Where("certificate_id = ? AND name = ?", cert.ID, info.CommonName).Count(&count)
	if count == 0 {
		s.DB.Create(&model.CertificateDomain{CertificateID: cert.ID, DomainID: &domainID, Name: info.CommonName})
	}
	s.DB.Model(&model.Domain{}).Where("id = ?", domainID).Update("last_cert_id", cert.ID)
	return &cert
}

func (s *Scheduler) notifyExpiry(userID uint, cert *model.Certificate, days int) {
	title := fmt.Sprintf("[CertPilot] 证书 %s 将于 %d 天后过期", cert.CommonName, days)
	content := fmt.Sprintf("域名 %s 的证书（颁发者：%s）将于 %s 过期，剩余 %d 天。", cert.CommonName, cert.Issuer, cert.NotAfter.Format("2006-01-02"), days)
	s.deliver(userID, &cert.ID, "expiry_remind", title, content)
}

// deliver 向用户所有启用的渠道发送通知并落日志
func (s *Scheduler) deliver(userID uint, certID *uint, typ, title, content string) {
	var channels []model.NotificationChannel
	s.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&channels)
	for _, ch := range channels {
		status, errMsg := "ok", ""
		if ch.Type == "email" {
			cfg := s.decodedConfig(ch.ConfigEnc)
			m := *s.Mailer
			if cfg.To != "" {
				err := m.Send(SplitCSV(cfg.To), title, content)
				if err != nil {
					status, errMsg = "fail", err.Error()
				}
			}
		} else {
			cfg := s.decodedConfig(ch.ConfigEnc)
			if err := SendToChannel(ch.Type, cfg, title, content); err != nil {
				status, errMsg = "fail", err.Error()
			}
		}
		s.DB.Create(&model.NotificationLog{
			UserID: userID, ChannelID: ch.ID, CertID: certID,
			Type: typ, Title: title, Content: content, Status: status, Error: errMsg,
		})
	}
}

func (s *Scheduler) decodedConfig(enc string) ChannelConfig {
	var cfg ChannelConfig
	if plain, err := Decrypt(s.Cfg.MasterKey, enc); err == nil {
		_ = jsonUnmarshal(plain, &cfg)
	}
	return cfg
}
