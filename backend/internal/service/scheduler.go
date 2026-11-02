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

// NextReminder 判定本轮应发送的提醒。
//
// left 为证书剩余天数，level 为已发送的档数水位（按阈值从大到小已发送的数量）。
// 返回值是本轮应发送后的新水位：命中的阈值数量 > 已发送水位时发送并推进水位——
// 某天漏跑后下次运行仍会补上提醒，而同一天内重启服务不会重复发送。
// 返回 0 表示本轮无需提醒。
func NextReminder(thresholds []int, left, level int) int {
	eligible := 0
	for _, days := range thresholds {
		if left <= days {
			eligible++
		}
	}
	if eligible > level {
		return eligible
	}
	return 0
}

// Run 执行一轮巡检：重新检测所有域名的 HTTPS 证书，并对命中阈值的证书发送提醒。
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
		left := int(time.Until(cert.NotAfter).Hours() / 24)
		if days := NextReminder(s.Cfg.ExpiryRemindDays, left, cert.NotifiedLevel); days > 0 {
			s.notifyExpiry(d.UserID, cert, left)
			if err := s.DB.Model(cert).Update("notified_level", days).Error; err != nil {
				log.Println("[scheduler] update notified level:", err)
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

// notifyExpiry 发送到期提醒。left 可能为负（已过期）。
func (s *Scheduler) notifyExpiry(userID uint, cert *model.Certificate, left int) {
	var title, content string
	if left < 0 {
		title = fmt.Sprintf("[CertPilot] 证书 %s 已过期 %d 天", cert.CommonName, -left)
		content = fmt.Sprintf("域名 %s 的证书（颁发者：%s）已于 %s 过期，请尽快更换！", cert.CommonName, cert.Issuer, cert.NotAfter.Format("2006-01-02"))
	} else {
		title = fmt.Sprintf("[CertPilot] 证书 %s 将于 %d 天后过期", cert.CommonName, left)
		content = fmt.Sprintf("域名 %s 的证书（颁发者：%s）将于 %s 过期，剩余 %d 天。", cert.CommonName, cert.Issuer, cert.NotAfter.Format("2006-01-02"), left)
	}
	s.deliver(userID, &cert.ID, "expiry_remind", title, content)
}

// deliver 向用户所有启用的渠道发送通知并落日志
func (s *Scheduler) deliver(userID uint, certID *uint, typ, title, content string) {
	var channels []model.NotificationChannel
	s.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&channels)
	for _, ch := range channels {
		status, errMsg := "ok", ""
		cfg := s.decodedConfig(ch.ConfigEnc)
		if ch.Type == "email" {
			m := *s.Mailer
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
