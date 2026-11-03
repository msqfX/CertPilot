package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/msqfX/CertPilot/backend/internal/model"
)

// 订单执行与自动续期。

func truncErr(s string) string {
	b := []byte(s)
	if len(b) > 500 {
		b = b[:500]
	}
	return string(b)
}

// ExecuteOrder 执行订单并回写状态（handler 异步调用）；签发结果发通知。
func (e *ACMEngine) ExecuteOrder(orderID uint) {
	var order model.CertificateOrder
	if err := e.DB.First(&order, orderID).Error; err != nil {
		log.Println("[acme] order not found:", orderID)
		return
	}
	cert, err := e.RunOrder(&order)
	if err != nil {
		e.DB.Model(&order).Updates(map[string]any{"status": "fail", "error": truncErr(err.Error())})
		var domains []string
		_ = json.Unmarshal([]byte(order.Domains), &domains)
		e.Notifier.Notify(order.UserID, nil, "issue_fail",
			fmt.Sprintf("[CertPilot] 证书签发失败：%s", domains),
			fmt.Sprintf("域名 %s 签发失败：%s", domains, err.Error()))
		return
	}
	e.DB.Model(&order).Updates(map[string]any{"status": "success", "certificate_id": cert.ID})
	e.Notifier.Notify(order.UserID, &cert.ID, "issue_ok",
		fmt.Sprintf("[CertPilot] 证书签发成功：%s", cert.CommonName),
		fmt.Sprintf("域名 %s 签发成功，有效期至 %s。", cert.CommonName, cert.NotAfter.Format("2006-01-02")))
}

// RunRenewals 扫描需要自动续期的证书（source=acme、auto_renew、剩余 <=30 天）。
// 成功：新证书替换旧证书（软删旧记录、域名指向新证书）；失败：发续期失败告警。
func (e *ACMEngine) RunRenewals() {
	var certs []model.Certificate
	cutoff := time.Now().AddDate(0, 0, 30)
	e.DB.Where("source = ? AND auto_renew = ? AND not_after <= ?", "acme", true, cutoff).Find(&certs)
	for i := range certs {
		cert := &certs[i]
		if cert.OrderID == nil {
			continue
		}
		var order model.CertificateOrder
		if err := e.DB.First(&order, *cert.OrderID).Error; err != nil {
			log.Println("[renew] original order missing for cert", cert.ID)
			continue
		}
		newOrder := model.CertificateOrder{
			UserID: cert.UserID, Domains: order.Domains,
			AccountID: order.AccountID, ProviderID: order.ProviderID,
			Status: "pending", Auto: true,
		}
		if err := e.DB.Create(&newOrder).Error; err != nil {
			log.Println("[renew] create order:", err)
			continue
		}

		newCert, err := e.RunOrder(&newOrder)
		if err != nil {
			e.DB.Model(&newOrder).Updates(map[string]any{"status": "fail", "error": truncErr(err.Error())})
			var domains []string
			_ = json.Unmarshal([]byte(order.Domains), &domains)
			e.Notifier.Notify(cert.UserID, &cert.ID, "renew_fail",
				fmt.Sprintf("[CertPilot] 证书续期失败：%s", cert.CommonName),
				fmt.Sprintf("域名 %s 续期失败：%s\n旧证书将于 %s 过期，请尽快手动处理。", domains, err.Error(), cert.NotAfter.Format("2006-01-02")))
			continue
		}

		e.DB.Model(&newOrder).Updates(map[string]any{"status": "success", "certificate_id": newCert.ID})
		oldID := cert.ID
		e.DB.Delete(cert) // 软删旧证书
		e.DB.Model(&model.Domain{}).Where("last_cert_id = ?", oldID).Update("last_cert_id", newCert.ID)
		var domains []string
		_ = json.Unmarshal([]byte(order.Domains), &domains)
		e.Notifier.Notify(cert.UserID, &newCert.ID, "renew_ok",
			fmt.Sprintf("[CertPilot] 证书续期成功：%s", newCert.CommonName),
			fmt.Sprintf("域名 %s 续期成功，新证书有效期至 %s。", domains, newCert.NotAfter.Format("2006-01-02")))
	}
}
