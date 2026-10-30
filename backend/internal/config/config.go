package config

import (
	"os"
)

// Config 全局配置，全部从环境变量读取。
type Config struct {
	Port string

	DBDSN string

	JWTSecret string
	JWTExpire int // 小时

	// AES-GCM 主密钥（32 字节 hex，64 个 hex 字符），用于加密私钥/SMTP 凭证等敏感数据
	MasterKey string

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// 到期提醒阈值（天），逗号分隔
	ExpiryRemindDays []int
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() *Config {
	c := &Config{
		Port:      getEnv("CERTPILOT_PORT", "8080"),
		DBDSN:     getEnv("CERTPILOT_DB_DSN", "root:root@tcp(127.0.0.1:3306)/certpilot?charset=utf8mb4&parseTime=True&loc=Local"),
		JWTSecret: getEnv("CERTPILOT_JWT_SECRET", "dev-only-secret-change-me"),
		JWTExpire: 72,
		MasterKey: getEnv("CERTPILOT_MASTER_KEY", ""),

		SMTPHost:     getEnv("CERTPILOT_SMTP_HOST", ""),
		SMTPUser:     getEnv("CERTPILOT_SMTP_USER", ""),
		SMTPPassword: getEnv("CERTPILOT_SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("CERTPILOT_SMTP_FROM", ""),
	}
	switch getEnv("CERTPILOT_SMTP_PORT", "465") {
	case "587":
		c.SMTPPort = 587
	default:
		c.SMTPPort = 465
	}
	c.ExpiryRemindDays = []int{30, 15, 7, 3}
	return c
}
