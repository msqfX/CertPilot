package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/model"
	"github.com/msqfX/CertPilot/backend/internal/service"
)

type ACMEHandler struct {
	DB     *gorm.DB
	Cfg    *config.Config
	Engine *service.ACMEngine
}

// ---------- ACME 账户 ----------

type accountResp struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	CA    string `json:"ca"`
	RegURI string `json:"reg_uri"`
}

func (h *ACMEHandler) ListAccounts(c *gin.Context) {
	uid := c.GetUint("uid")
	var accs []model.AcmeAccount
	h.DB.Where("user_id = ?", uid).Find(&accs)
	out := make([]accountResp, 0, len(accs))
	for _, a := range accs {
		out = append(out, accountResp{ID: a.ID, Email: a.Email, CA: a.CA, RegURI: a.RegURI})
	}
	c.JSON(http.StatusOK, out)
}

func (h *ACMEHandler) CreateAccount(c *gin.Context) {
	uid := c.GetUint("uid")
	if !service.MasterKeyOK(h.Cfg.MasterKey) {
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "服务端未配置 CERTPILOT_MASTER_KEY，无法注册 ACME 账户"})
		return
	}
	var req struct {
		Email string `json:"email" binding:"required,email"`
		CA    string `json:"ca" binding:"omitempty,oneof=letsencrypt letsencrypt-staging"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CA == "" {
		req.CA = "letsencrypt"
	}
	acc, err := h.Engine.RegisterAccount(uid, req.Email, req.CA)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	h.DB.Create(&model.OperationLog{UserID: uid, Action: "acme.account.create", Target: req.Email, IP: c.ClientIP()})
	c.JSON(http.StatusOK, accountResp{ID: acc.ID, Email: acc.Email, CA: acc.CA, RegURI: acc.RegURI})
}

// ---------- DNS 服务商 ----------

func (h *ACMEHandler) ListProviders(c *gin.Context) {
	uid := c.GetUint("uid")
	var ps []model.DNSProvider
	h.DB.Where("user_id = ?", uid).Find(&ps)
	out := make([]gin.H, 0, len(ps))
	for _, p := range ps {
		out = append(out, gin.H{"id": p.ID, "provider": p.Provider, "name": p.Name})
	}
	c.JSON(http.StatusOK, out)
}

func (h *ACMEHandler) CreateProvider(c *gin.Context) {
	uid := c.GetUint("uid")
	if !service.MasterKeyOK(h.Cfg.MasterKey) {
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "服务端未配置 CERTPILOT_MASTER_KEY，无法存储 DNS 凭证"})
		return
	}
	var req struct {
		Provider string          `json:"provider" binding:"required,oneof=cloudflare alidns tencentcloud"`
		Name     string          `json:"name"`
		Config   json.RawMessage `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enc, err := service.Encrypt(h.Cfg.MasterKey, string(req.Config))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	p := model.DNSProvider{UserID: uid, Provider: req.Provider, Name: req.Name, ConfigEnc: enc}
	if err := h.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": p.ID, "provider": p.Provider, "name": p.Name})
}

func (h *ACMEHandler) DeleteProvider(c *gin.Context) {
	uid := c.GetUint("uid")
	res := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).Delete(&model.DNSProvider{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- 证书订单 ----------

type orderResp struct {
	model.CertificateOrder
	DomainList []string `json:"domain_list"`
	Provider   string   `json:"provider"`
	Account    string   `json:"account"`
}

func (h *ACMEHandler) ListOrders(c *gin.Context) {
	uid := c.GetUint("uid")
	var orders []model.CertificateOrder
	h.DB.Where("user_id = ?", uid).Order("created_at DESC").Limit(50).Find(&orders)
	out := make([]orderResp, 0, len(orders))
	for _, o := range orders {
		r := orderResp{CertificateOrder: o}
		_ = json.Unmarshal([]byte(o.Domains), &r.DomainList)
		var p model.DNSProvider
		if err := h.DB.First(&p, o.ProviderID).Error; err == nil {
			r.Provider = p.Name
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, out)
}

// CreateOrder 创建签发订单并异步执行
func (h *ACMEHandler) CreateOrder(c *gin.Context) {
	uid := c.GetUint("uid")
	var req struct {
		Domains     []string `json:"domains" binding:"required,min=1"`
		AccountID   uint     `json:"account_id" binding:"required"`
		ProviderID  uint     `json:"provider_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 清洗域名
	cleaned := make([]string, 0, len(req.Domains))
	for _, d := range req.Domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if !validDomain(d) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "非法域名: " + d})
			return
		}
		cleaned = append(cleaned, d)
	}
	if len(cleaned) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一个域名"})
		return
	}
	// 归属校验
	var accCount, provCount int64
	h.DB.Model(&model.AcmeAccount{}).Where("id = ? AND user_id = ?", req.AccountID, uid).Count(&accCount)
	h.DB.Model(&model.DNSProvider{}).Where("id = ? AND user_id = ?", req.ProviderID, uid).Count(&provCount)
	if accCount == 0 || provCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ACME 账户或 DNS 服务商不存在"})
		return
	}

	domainsJSON, _ := json.Marshal(cleaned)
	order := model.CertificateOrder{
		UserID: uid, Domains: string(domainsJSON),
		AccountID: req.AccountID, ProviderID: req.ProviderID, Status: "pending",
	}
	if err := h.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.DB.Create(&model.OperationLog{UserID: uid, Action: "cert.order.create", Target: strings.Join(cleaned, ","), IP: c.ClientIP()})
	go h.Engine.ExecuteOrder(order.ID)
	c.JSON(http.StatusOK, gin.H{"id": order.ID, "status": "pending"})
}
