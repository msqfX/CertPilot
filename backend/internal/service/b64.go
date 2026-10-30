package service

import "encoding/base64"

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
