package indirectmerchant

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

// SensitiveFieldConfig 为间连商户敏感字段提供平台证书与商户私钥。
// 字段含义：平台证书用于上送加密，商户私钥用于下行解密；只用于本地处理。
type SensitiveFieldConfig struct {
	PlatformCertificate *x509.Certificate
	MerchantPrivateKey  *rsa.PrivateKey
}

func (a *ApiService) encryptIdentification(req CreateApplyRequest) (CreateApplyRequest, error) {
	if req.IdentificationInfo == nil {
		return req, nil
	}
	info := *req.IdentificationInfo
	req.IdentificationInfo = &info
	if info.IdentificationName == "" && info.IdentificationNumber == "" {
		return req, nil
	}
	if a.SensitiveFields == nil || a.SensitiveFields.PlatformCertificate == nil {
		return req, fmt.Errorf("platform certificate is required to encrypt identification fields")
	}
	cert := a.SensitiveFields.PlatformCertificate
	key, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return req, fmt.Errorf("identification fields require an RSA platform certificate")
	}
	serial := fmt.Sprintf("%X", cert.SerialNumber)
	if req.DouyinpaySerial != "" && req.DouyinpaySerial != serial {
		return req, fmt.Errorf("Douyinpay-Serial must match the encryption certificate")
	}
	req.DouyinpaySerial = serial
	var err error
	if info.IdentificationName, err = encryptField(info.IdentificationName, key); err != nil {
		return req, err
	}
	if info.IdentificationNumber, err = encryptField(info.IdentificationNumber, key); err != nil {
		return req, err
	}
	return req, nil
}

func encryptField(value string, key *rsa.PublicKey) (string, error) {
	if value == "" {
		return value, nil
	}
	data, err := rsa.EncryptPKCS1v15(rand.Reader, key, []byte(value))
	if err != nil {
		return "", fmt.Errorf("encrypt identification field: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func decryptField(value string, key *rsa.PrivateKey) (string, error) {
	if value == "" {
		return value, nil
	}
	if key == nil {
		return "", fmt.Errorf("merchant private key is required to decrypt identification fields")
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("decode identification ciphertext: %w", err)
	}
	plain, err := rsa.DecryptPKCS1v15(rand.Reader, key, data)
	if err != nil {
		return "", fmt.Errorf("decrypt identification field: %w", err)
	}
	return string(plain), nil
}
