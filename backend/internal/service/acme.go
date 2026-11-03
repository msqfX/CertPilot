package service

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/tencentcloud"
	"github.com/go-acme/lego/v4/registration"
	"gorm.io/gorm"

	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/model"
)

// ACME 引擎：基于 Lego 实现证书申请（DNS-01）与自动续期。

const (
	CAProductionURL = "https://acme-v02.api.letsencrypt.org/directory"
	CAStagingURL    = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

func CAURL(ca string) string {
	if ca == "letsencrypt-staging" {
		return CAStagingURL
	}
	return CAProductionURL
}

// ---------- lego user ----------

type legoUser struct {
	email string
	reg   *registration.Resource
	key   crypto.PrivateKey
}

func (u *legoUser) GetEmail() string                        { return u.email }
func (u *legoUser) GetRegistration() *registration.Resource { return u.reg }
func (u *legoUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// ---------- 账户密钥 ----------

// GenerateAccountKeyPEM 生成 ECDSA P-256 账户私钥的 PEM
func GenerateAccountKeyPEM() (string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})), nil
}

func parseAccountKey(pemStr string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid account key PEM")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

// ---------- DNS 服务商 ----------

type cloudflareCfg struct {
	AuthToken string `json:"auth_token"`
}
type alidnsCfg struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}
type tencentCfg struct {
	SecretID  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
}

// ResolveDNSProvider 根据存储的加密凭证还原 lego DNS provider
func ResolveDNSProvider(providerType, configEnc, masterKey string) (challenge.Provider, error) {
	plain, err := Decrypt(masterKey, configEnc)
	if err != nil {
		return nil, fmt.Errorf("解密 DNS 凭证失败: %w", err)
	}
	switch providerType {
	case "cloudflare":
		var c cloudflareCfg
		if err := json.Unmarshal([]byte(plain), &c); err != nil {
			return nil, err
		}
		return cloudflare.NewDNSProviderConfig(&cloudflare.Config{
			AuthToken:          c.AuthToken,
			PropagationTimeout: 5 * time.Minute,
			PollingInterval:    10 * time.Second,
		})
	case "alidns":
		var c alidnsCfg
		if err := json.Unmarshal([]byte(plain), &c); err != nil {
			return nil, err
		}
		return alidns.NewDNSProviderConfig(&alidns.Config{
			APIKey:             c.AccessKey,
			SecretKey:          c.SecretKey,
			RegionID:           "cn-hangzhou",
			PropagationTimeout: 5 * time.Minute,
			PollingInterval:    10 * time.Second,
		})
	case "tencentcloud":
		var c tencentCfg
		if err := json.Unmarshal([]byte(plain), &c); err != nil {
			return nil, err
		}
		return tencentcloud.NewDNSProviderConfig(&tencentcloud.Config{
			SecretID:           c.SecretID,
			SecretKey:          c.SecretKey,
			PropagationTimeout: 5 * time.Minute,
			PollingInterval:    10 * time.Second,
		})
	default:
		return nil, fmt.Errorf("不支持的 DNS 服务商: %s", providerType)
	}
}

// ---------- 引擎 ----------

type ACMEngine struct {
	DB       *gorm.DB
	Cfg      *config.Config
	Notifier *Notifier
}

// RegisterAccount 注册 ACME 账户（每用户每 CA 一个）
func (e *ACMEngine) RegisterAccount(userID uint, email, ca string) (*model.AcmeAccount, error) {
	var count int64
	e.DB.Model(&model.AcmeAccount{}).Where("user_id = ? AND ca = ?", userID, ca).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("该 CA 下已有账户，一个用户每个 CA 只需一个账户")
	}
	keyPEM, err := GenerateAccountKeyPEM()
	if err != nil {
		return nil, err
	}
	key, err := parseAccountKey(keyPEM)
	if err != nil {
		return nil, err
	}
	user := &legoUser{email: email, key: key}
	legoCfg := lego.NewConfig(user)
	legoCfg.CADirURL = CAURL(ca)
	client, err := lego.NewClient(legoCfg)
	if err != nil {
		return nil, err
	}
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, fmt.Errorf("ACME 账户注册失败: %w", err)
	}
	keyEnc, err := Encrypt(e.Cfg.MasterKey, keyPEM)
	if err != nil {
		return nil, err
	}
	acc := &model.AcmeAccount{
		UserID: userID, Email: email, CA: ca,
		PrivateKeyEnc: keyEnc, RegURI: reg.URI,
	}
	if err := e.DB.Create(acc).Error; err != nil {
		return nil, err
	}
	return acc, nil
}

// buildClient 用已注册的账户构造 lego client
func (e *ACMEngine) buildClient(acc *model.AcmeAccount) (*lego.Client, error) {
	keyPEM, err := Decrypt(e.Cfg.MasterKey, acc.PrivateKeyEnc)
	if err != nil {
		return nil, fmt.Errorf("解密 ACME 账户私钥失败: %w", err)
	}
	key, err := parseAccountKey(keyPEM)
	if err != nil {
		return nil, err
	}
	user := &legoUser{email: acc.Email, key: key, reg: &registration.Resource{URI: acc.RegURI}}
	legoCfg := lego.NewConfig(user)
	legoCfg.CADirURL = CAURL(acc.CA)
	legoCfg.Certificate.KeyType = certcrypto.EC256
	return lego.NewClient(legoCfg)
}

// RunOrder 执行一个签发订单（异步订单与自动续期共用）。
// 成功时创建证书记录并关联域名；返回创建的证书。
func (e *ACMEngine) RunOrder(order *model.CertificateOrder) (*model.Certificate, error) {
	var acc model.AcmeAccount
	if err := e.DB.First(&acc, order.AccountID).Error; err != nil {
		return nil, fmt.Errorf("ACME 账户不存在")
	}
	var provider model.DNSProvider
	if err := e.DB.First(&provider, order.ProviderID).Error; err != nil {
		return nil, fmt.Errorf("DNS 服务商配置不存在")
	}

	client, err := e.buildClient(&acc)
	if err != nil {
		return nil, err
	}
	dnsProvider, err := ResolveDNSProvider(provider.Provider, provider.ConfigEnc, e.Cfg.MasterKey)
	if err != nil {
		return nil, err
	}
	if err := client.Challenge.SetDNS01Provider(dnsProvider); err != nil {
		return nil, fmt.Errorf("配置 DNS-01 验证失败: %w", err)
	}

	var domains []string
	if err := json.Unmarshal([]byte(order.Domains), &domains); err != nil || len(domains) == 0 {
		return nil, fmt.Errorf("订单域名为空或格式错误")
	}

	res, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("签发失败: %w", err)
	}

	// 解析叶子证书
	info, err := ParsePEM(res.Certificate)
	if err != nil {
		return nil, fmt.Errorf("解析新证书失败: %w", err)
	}
	keyEnc, err := Encrypt(e.Cfg.MasterKey, string(res.PrivateKey))
	if err != nil {
		return nil, err
	}

	cert := &model.Certificate{
		UserID:      order.UserID,
		Name:        domains[0],
		CommonName:  info.CommonName,
		Issuer:      info.Issuer,
		NotBefore:   info.NotBefore,
		NotAfter:    info.NotAfter,
		Fingerprint: info.Fingerprint,
		PrivateKey:  keyEnc,
		ChainPEM:    string(res.Certificate),
		Source:      "acme",
		AutoRenew:   true,
		OrderID:     &order.ID,
	}
	if err := e.DB.Create(cert).Error; err != nil {
		return nil, err
	}
	for _, n := range info.DNSNames {
		var domainID *uint
		var d model.Domain
		if err := e.DB.Where("name = ?", n).First(&d).Error; err == nil {
			domainID = &d.ID
		}
		e.DB.Create(&model.CertificateDomain{CertificateID: cert.ID, DomainID: domainID, Name: n})
	}
	return cert, nil
}
