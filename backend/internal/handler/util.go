package handler

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func trimSpace(s string) string { return strings.TrimSpace(s) }

func trimPrefixes(s string) string {
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	if i := strings.Index(s, "/"); i > 0 {
		s = s[:i]
	}
	if i := strings.Index(s, ":"); i > 0 {
		s = s[:i]
	}
	return strings.ToLower(s)
}

var domainRe = regexp.MustCompile(`^(\*\.)?([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$`)

func validDomain(s string) bool { return domainRe.MatchString(s) }

func splitLines(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' })
	var out []string
	for _, f := range fields {
		if strings.TrimSpace(f) != "" {
			out = append(out, f)
		}
	}
	return out
}

// splitPemCertKey 将 PEM 文本拆分为证书链与私钥，并校验两者匹配
func splitPemCertKey(data []byte) (chainPEM string, keyPEM string, err error) {
	var chain, key strings.Builder
	rest := data
	var leaf *x509.Certificate
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		pemBytes := pem.EncodeToMemory(block)
		switch block.Type {
		case "CERTIFICATE":
			chain.Write(pemBytes)
			if leaf == nil {
				c, e := x509.ParseCertificate(block.Bytes)
				if e != nil {
					return "", "", fmt.Errorf("parse cert: %w", e)
				}
				leaf = c
			}
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			key.Write(pemBytes)
		default:
			// 忽略其他块
		}
	}
	if leaf == nil {
		return "", "", errors.New("no CERTIFICATE block found")
	}
	if key.Len() == 0 {
		return "", "", errors.New("no PRIVATE KEY block found")
	}
	// 校验证书与私钥匹配
	if _, err := tls.X509KeyPair([]byte(chain.String()), []byte(key.String())); err != nil {
		return "", "", fmt.Errorf("cert and key do not match: %w", err)
	}
	return chain.String(), key.String(), nil
}
