package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Email        string         `gorm:"size:191;uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Nickname     string         `gorm:"size:64" json:"nickname"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Domain struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`
	Name         string         `gorm:"size:255;uniqueIndex;not null" json:"name"` // 如 example.com
	Remark       string         `gorm:"size:255" json:"remark"`
	LastCheckAt  *time.Time     `json:"last_check_at"`
	LastCertID   *uint          `json:"last_cert_id"` // 最近一次检测到的证书
	LastError    string         `gorm:"size:512" json:"last_error"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Certificate struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	Name        string         `gorm:"size:128" json:"name"` // 展示名，默认取主域名
	CommonName  string         `gorm:"size:255" json:"common_name"`
	Issuer      string         `gorm:"size:255" json:"issuer"`
	NotBefore   time.Time      `json:"not_before"`
	NotAfter    time.Time      `json:"not_after"`
	Fingerprint string         `gorm:"size:128;index" json:"fingerprint"` // SHA-256，用于识别是否轮换
	PrivateKey  string         `gorm:"type:text" json:"-"`                // AES-GCM 加密存储
	ChainPEM    string         `gorm:"type:mediumtext" json:"-"`          // 证书链（含叶子）
	Source      string         `gorm:"size:32;default:upload" json:"source"` // upload / acme
	AutoRenew   bool           `gorm:"default:false" json:"auto_renew"`   // V1.5 起生效
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// CertificateDomain 证书与域名关联（SAN 多域名）
type CertificateDomain struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CertificateID uint      `gorm:"index;not null" json:"certificate_id"`
	DomainID      *uint     `gorm:"index" json:"domain_id"` // 可为空：证书里的域名不一定已录入
	Name          string    `gorm:"size:255;not null" json:"name"`
	CreatedAt     time.Time `json:"created_at"`
}

// NotificationChannel 通知渠道（email / webhook / wecom），凭证加密存储
type NotificationChannel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Type      string         `gorm:"size:32;not null" json:"type"` // email / webhook / wecom
	Name      string         `gorm:"size:64" json:"name"`
	ConfigEnc string         `gorm:"type:text;not null" json:"-"` // AES-GCM 加密的 JSON 配置
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// NotificationLog 通知发送记录
type NotificationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	ChannelID uint      `json:"channel_id"`
	CertID    *uint     `json:"cert_id"`
	Type      string    `gorm:"size:32" json:"type"` // expiry_remind / renew_ok / renew_fail
	Title     string    `gorm:"size:255" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	Status    string    `gorm:"size:16" json:"status"` // ok / fail
	Error     string    `gorm:"size:512" json:"error"`
	CreatedAt time.Time `json:"created_at"`
}

// OperationLog 操作日志
type OperationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Action    string    `gorm:"size:64" json:"action"`   // 如 domain.create / cert.upload
	Target    string    `gorm:"size:255" json:"target"`  // 操作对象
	Detail    string    `gorm:"type:text" json:"detail"`
	IP        string    `gorm:"size:64" json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}
