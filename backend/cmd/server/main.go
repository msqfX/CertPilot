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

	db, err := database.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	mailer := &service.Mailer{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort,
		User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
	}
	scheduler := &service.Scheduler{DB: db, Cfg: cfg, Mailer: mailer}
	scheduler.Start()

	r := router.Setup(db, cfg)
	log.Printf("[server] CertPilot API listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
