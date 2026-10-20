package app

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"strings"
)

// CertHint is the hint for a server certificate verification error (C10 v5).
const CertHint = "server certificate is not trusted — set the CA in http.ca_file"

// IsCertError reports a server certificate verification error (including one that lost its type: matched by the "x509:" text).
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
