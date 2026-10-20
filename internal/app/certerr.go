package app

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"strings"
)

// CertHint — подсказка при ошибке проверки сертификата сервера (C10 v5).
const CertHint = "сертификат сервера не доверен — укажите CA в http.ca_file"

// IsCertError — ошибка проверки сертификата сервера (в т.ч. потерявшая тип: по тексту "x509:").
func IsCertError(err error) bool {
	if err == nil {
		return false
	}
	var (
		ua  x509.UnknownAuthorityError
		ci  x509.CertificateInvalidError
		hn  x509.HostnameError
		tlv *tls.CertificateVerificationError
	)
	return errors.As(err, &ua) || errors.As(err, &ci) || errors.As(err, &hn) ||
		errors.As(err, &tlv) || strings.Contains(err.Error(), "x509:")
}
