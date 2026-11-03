package model

import (
	"time"

	"gorm.io/gorm"
)

// AcmeAccount ACME 账户（每用户每 CA 一个），账户私钥 AES-GCM 加密存储
type AcmeAccount struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	UserID        uint           `gorm:"index;not null" json:"user_id"`
	Email         string         `gorm:"size:191;not null" json:"email"`
	CA            string         `gorm:"size:32;default:letsencrypt" json:"ca"` // letsencrypt / letsencrypt-staging
	PrivateKeyEnc string         `gorm:"type:text;not null" json:"-"`
	RegURI        string         `gorm:"size:255" json:"reg_uri"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// DNSProvider DNS 服务商凭证（cloudflare / alidns / tencentcloud），凭证 AES-GCM 加密存储
type DNSProvider struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Provider  string         `gorm:"size:32;not null" json:"provider"`
	Name      string         `gorm:"size:64" json:"name"`
	ConfigEnc string         `gorm:"type:text;not null" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// CertificateOrder 证书签发订单（异步执行），成功后关联证书
type CertificateOrder struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	UserID        uint           `gorm:"index;not null" json:"user_id"`
	Domains       string         `gorm:"type:text;not null" json:"domains"` // JSON 数组
	AccountID     uint           `json:"account_id"`
	ProviderID    uint           `json:"provider_id"`
	Status        string         `gorm:"size:16;default:pending;index" json:"status"` // pending / success / fail
	Error         string         `gorm:"size:512" json:"error"`
	CertificateID *uint          `gorm:"index" json:"certificate_id"`
	Auto          bool           `gorm:"default:false" json:"auto"` // 自动续期产生的订单
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
