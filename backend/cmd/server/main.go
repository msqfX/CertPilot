package main

import (
	"log"

	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/database"
	"github.com/msqfX/CertPilot/backend/internal/router"
	"github.com/msqfX/CertPilot/backend/internal/service"
)

func main() {
	cfg := config.Load()

	if cfg.MasterKey == "" {
		log.Println("[warn] CERTPILOT_MASTER_KEY 未配置：证书上传与通知渠道功能不可用，生成方式见 backend/.env.example")
	}
	if !cfg.AllowRegister {
		log.Println("[info] 注册已关闭（CERTPILOT_ALLOW_REGISTER=false）")
	}

	db, err := database.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	mailer := &service.Mailer{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort,
		User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
	}
	notifier := &service.Notifier{DB: db, Cfg: cfg, Mailer: mailer}
	acmEngine := &service.ACMEngine{DB: db, Cfg: cfg, Notifier: notifier}
	scheduler := &service.Scheduler{DB: db, Cfg: cfg, Mailer: mailer, ACM: acmEngine}
	scheduler.Start()

	r := router.Setup(db, cfg, acmEngine)
	log.Printf("[server] CertPilot API listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
