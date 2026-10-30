package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/model"
	"github.com/msqfX/CertPilot/backend/internal/service"
)

type CertificateHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

type certResp struct {
	model.Certificate
	Domains     []string `json:"domains"`
	DaysLeft    int      `json:"days_left"`
	HasKey      bool     `json:"has_key"`
	Expired     bool     `json:"expired"`
	ExpiringSoon bool    `json:"expiring_soon"` // 剩余 <=30 天
}

func (h *CertificateHandler) List(c *gin.Context) {
	uid := c.GetUint("uid")
	var certs []model.Certificate
	h.DB.Where("user_id = ?", uid).Order("not_after ASC").Find(&certs)
	out := make([]certResp, 0, len(certs))
	for _, ct := range certs {
		out = append(out, h.buildResp(ct))
	}
	c.JSON(http.StatusOK, out)
}

// Upload 上传已有证书（PEM 文本：证书链 + 私钥），私钥 AES-GCM 加密落库
func (h *CertificateHandler) Upload(c *gin.Context) {
	uid := c.GetUint("uid")
	var req struct {
		Name string `json:"name"`
		Pem  string `json:"pem" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	info, err := service.ParsePEM([]byte(req.Pem))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "证书解析失败: " + err.Error()})
		return
	}
	chain, key, err := splitPemCertKey([]byte(req.Pem))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	keyEnc, err := service.Encrypt(h.Cfg.MasterKey, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	name := req.Name
	if name == "" {
		name = info.CommonName
	}
	ct := model.Certificate{
		UserID: uid, Name: name, CommonName: info.CommonName, Issuer: info.Issuer,
		NotBefore: info.NotBefore, NotAfter: info.NotAfter,
		Fingerprint: info.Fingerprint, PrivateKey: keyEnc, ChainPEM: chain, Source: "upload",
	}
	if err := h.DB.Create(&ct).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 关联 SAN 域名
	for _, n := range info.DNSNames {
		var domainID *uint
		var d model.Domain
		if err := h.DB.Where("name = ?", n).First(&d).Error; err == nil {
			domainID = &d.ID
		}
		h.DB.Create(&model.CertificateDomain{CertificateID: ct.ID, DomainID: domainID, Name: n})
	}
	h.DB.Create(&model.OperationLog{UserID: uid, Action: "cert.upload", Target: info.CommonName, IP: c.ClientIP()})
	c.JSON(http.StatusOK, h.buildResp(ct))
}

// Download 下载证书链或私钥（?type=pem|key），私钥解密后返回
func (h *CertificateHandler) Download(c *gin.Context) {
	uid := c.GetUint("uid")
	var ct model.Certificate
	if err := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).First(&ct).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}
	typ := c.Query("type")
	var body, filename string
	switch typ {
	case "key":
		if ct.PrivateKey == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "no private key stored"})
			return
		}
		plain, err := service.Decrypt(h.Cfg.MasterKey, ct.PrivateKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		body, filename = plain, ct.CommonName+".key"
	default:
		body, filename = ct.ChainPEM, ct.CommonName+".pem"
	}
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/x-pem-file", []byte(body))
}

func (h *CertificateHandler) Delete(c *gin.Context) {
	uid := c.GetUint("uid")
	res := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).Delete(&model.Certificate{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}
	h.DB.Create(&model.OperationLog{UserID: uid, Action: "cert.delete", Target: c.Param("id"), IP: c.ClientIP()})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *CertificateHandler) buildResp(ct model.Certificate) certResp {
	var names []string
	h.DB.Model(&model.CertificateDomain{}).Where("certificate_id = ?", ct.ID).Pluck("name", &names)
	days := int(time.Until(ct.NotAfter).Hours() / 24)
	return certResp{
		Certificate: ct, Domains: names, DaysLeft: days,
		HasKey: ct.PrivateKey != "", Expired: days < 0, ExpiringSoon: days >= 0 && days <= 30,
	}
}
