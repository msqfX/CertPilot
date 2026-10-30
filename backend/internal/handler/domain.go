package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/msqfX/CertPilot/backend/internal/model"
	"github.com/msqfX/CertPilot/backend/internal/service"
)

type DomainHandler struct {
	DB *gorm.DB
}

type domainResp struct {
	model.Domain
	Cert *service.CertInfo `json:"cert,omitempty"`
}

func (h *DomainHandler) List(c *gin.Context) {
	uid := c.GetUint("uid")
	var domains []model.Domain
	h.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&domains)
	out := make([]domainResp, 0, len(domains))
	for _, d := range domains {
		r := domainResp{Domain: d}
		if d.LastCertID != nil {
			var cert model.Certificate
			if err := h.DB.First(&cert, *d.LastCertID).Error; err == nil {
				r.Cert = &service.CertInfo{
					CommonName: cert.CommonName, Issuer: cert.Issuer,
					NotBefore: cert.NotBefore, NotAfter: cert.NotAfter,
					Fingerprint: cert.Fingerprint,
					DaysLeft:    int(time.Until(cert.NotAfter).Hours() / 24),
				}
			}
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, out)
}

// Create 添加域名（可选立即检测）
func (h *DomainHandler) Create(c *gin.Context) {
	uid := c.GetUint("uid")
	var req struct {
		Name   string `json:"name" binding:"required"`
		Remark string `json:"remark"`
		Check  bool   `json:"check"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d := model.Domain{UserID: uid, Name: normalizeDomain(req.Name), Remark: req.Remark}
	if d.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain"})
		return
	}
	var count int64
	h.DB.Model(&model.Domain{}).Where("name = ?", d.Name).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "域名已存在"})
		return
	}
	if err := h.DB.Create(&d).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if req.Check {
		h.checkOne(&d)
		h.DB.Save(&d)
	}
	c.JSON(http.StatusOK, d)
}

// BatchImport 批量导入（一行一个域名）
func (h *DomainHandler) BatchImport(c *gin.Context) {
	uid := c.GetUint("uid")
	var req struct {
		Names string `json:"names" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, skipped := 0, 0
	for _, line := range splitLines(req.Names) {
		name := normalizeDomain(line)
		if name == "" {
			continue
		}
		var count int64
		h.DB.Model(&model.Domain{}).Where("name = ?", name).Count(&count)
		if count > 0 {
			skipped++
			continue
		}
		if err := h.DB.Create(&model.Domain{UserID: uid, Name: name}).Error; err == nil {
			created++
		}
	}
	c.JSON(http.StatusOK, gin.H{"created": created, "skipped": skipped})
}

// Check 手动触发一次 HTTPS 检测
func (h *DomainHandler) Check(c *gin.Context) {
	uid := c.GetUint("uid")
	var d model.Domain
	if err := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).First(&d).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	info, err := service.CheckHTTPS(d.Name, 443)
	now := time.Now()
	d.LastCheckAt = &now
	if err != nil {
		d.LastError = err.Error()
		h.DB.Save(&d)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "domain": d})
		return
	}
	d.LastError = ""
	h.DB.Save(&d)

	// 检测到的证书入库并关联
	var cert model.Certificate
	if err := h.DB.Where("fingerprint = ?", info.Fingerprint).First(&cert).Error; err != nil {
		cert = model.Certificate{
			UserID: uid, Name: info.CommonName, CommonName: info.CommonName,
			Issuer: info.Issuer, NotBefore: info.NotBefore, NotAfter: info.NotAfter,
			Fingerprint: info.Fingerprint, Source: "monitor",
		}
		h.DB.Create(&cert)
		h.DB.Create(&model.CertificateDomain{CertificateID: cert.ID, DomainID: &d.ID, Name: info.CommonName})
	}
	d.LastCertID = &cert.ID
	h.DB.Save(&d)
	c.JSON(http.StatusOK, gin.H{"domain": d, "cert": info})
}

func (h *DomainHandler) Delete(c *gin.Context) {
	uid := c.GetUint("uid")
	res := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).Delete(&model.Domain{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *DomainHandler) checkOne(d *model.Domain) {
	info, err := service.CheckHTTPS(d.Name, 443)
	now := time.Now()
	d.LastCheckAt = &now
	if err != nil {
		d.LastError = err.Error()
		return
	}
	d.LastError = ""
	_ = info
}

func normalizeDomain(s string) string {
	s = trimSpace(s)
	s = trimPrefixes(s)
	if !validDomain(s) {
		return ""
	}
	return s
}
