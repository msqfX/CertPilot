package service

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// 证书解析与 HTTPS 检测（V1.0 核心能力）。

// CertInfo 证书摘要信息
type CertInfo struct {
	CommonName  string    `json:"common_name"`
	DNSNames    []string  `json:"dns_names"`
	Issuer      string    `json:"issuer"`
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	Fingerprint string    `json:"fingerprint"`
	DaysLeft    int       `json:"days_left"`
}

// CheckHTTPS 拨测公网 HTTPS，返回叶子证书信息（监控路径，无需任何服务器权限）
func CheckHTTPS(host string, port int) (*CertInfo, error) {
	if port == 0 {
		port = 443
	}
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	d := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(d, "tcp", addr, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, errors.New("no certificate presented")
	}
	return FromX509(state.PeerCertificates[0]), nil
}

// FromX509 x509 证书转摘要
func FromX509(c *x509.Certificate) *CertInfo {
	sum := sha256.Sum256(c.Raw)
	return &CertInfo{
		CommonName:  c.Subject.CommonName,
		DNSNames:    c.DNSNames,
		Issuer:      issuerName(c),
		NotBefore:   c.NotBefore,
		NotAfter:    c.NotAfter,
		Fingerprint: hex.EncodeToString(sum[:]),
		DaysLeft:    int(time.Until(c.NotAfter).Hours() / 24),
	}
}

func issuerName(c *x509.Certificate) string {
	for _, org := range c.Issuer.Organization {
		if org != "" {
			return org
		}
	}
	return c.Issuer.CommonName
}

// ParsePEM 解析 PEM 证书（支持证书链，取第一个证书为叶子）
func ParsePEM(pemData []byte) (*CertInfo, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("no CERTIFICATE block found in PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	return FromX509(cert), nil
}

// MatchDomain 判断证书覆盖某域名（含泛域名）
func MatchDomain(info *CertInfo, domain string) bool {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	for _, n := range info.DNSNames {
		n = strings.ToLower(n)
		if n == domain {
			return true
		}
		if strings.HasPrefix(n, "*.") {
			suffix := strings.TrimPrefix(n, "*.")
			if strings.HasSuffix(domain, "."+suffix) {
				rest := strings.TrimSuffix(domain, "."+suffix)
				if !strings.Contains(rest, ".") { // *.a.com 只匹配一级子域
					return true
				}
			}
		}
	}
	return false
}
