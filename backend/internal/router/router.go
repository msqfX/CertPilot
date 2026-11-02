package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/handler"
	"github.com/msqfX/CertPilot/backend/internal/middleware"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)
	r.Use(middleware.CORS())
	middleware.JWTSecret = []byte(cfg.JWTSecret)

	auth := &handler.AuthHandler{DB: db, Cfg: cfg}
	domains := &handler.DomainHandler{DB: db}
	certs := &handler.CertificateHandler{DB: db, Cfg: cfg}
	dash := &handler.DashboardHandler{DB: db}
	notif := &handler.NotificationHandler{DB: db, Cfg: cfg}
	logs := &handler.LogHandler{DB: db}

	api := r.Group("/api/v1")
	{
		api.POST("/auth/register", auth.Register)
		api.POST("/auth/login", auth.Login)

		authed := api.Group("", middleware.JWTAuth())
		{
			authed.GET("/auth/me", auth.Me)

			authed.GET("/domains", domains.List)
			authed.POST("/domains", domains.Create)
			authed.POST("/domains/batch", domains.BatchImport)
			authed.POST("/domains/:id/check", domains.Check)
			authed.DELETE("/domains/:id", domains.Delete)

			authed.GET("/certificates", certs.List)
			authed.POST("/certificates", certs.Upload)
			authed.GET("/certificates/:id/download", certs.Download)
			authed.DELETE("/certificates/:id", certs.Delete)

			authed.GET("/dashboard/stats", dash.Stats)
			authed.GET("/dashboard/upcoming", dash.Upcoming)

			authed.GET("/notifications/channels", notif.ListChannels)
			authed.POST("/notifications/channels", notif.CreateChannel)
			authed.POST("/notifications/channels/:id/test", notif.TestChannel)
			authed.DELETE("/notifications/channels/:id", notif.DeleteChannel)
			authed.GET("/notifications/logs", notif.Logs)

			authed.GET("/operation-logs", logs.List)
		}
	}

	serveFrontend(r)
	return r
}

// serveFrontend 托管前端构建产物（web/dist），SPA 路由回退到 index.html。
// 目录不存在时跳过（纯 API 模式 / 前后端分离部署）。
func serveFrontend(r *gin.Engine) {
	dist := "./web/dist"
	if _, err := os.Stat(dist); err != nil {
		return
	}
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		full := filepath.Join(dist, filepath.Clean("/"+p))
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			c.File(full)
			return
		}
		c.File(filepath.Join(dist, "index.html"))
	})
}
