package app

import (
	"crypto/x509"
	"errors"
	"fmt"
	"testing"
)

func TestContractCertError(t *testing.T) {
	yes := []error{
		x509.UnknownAuthorityError{},
		fmt.Errorf("wrap: %w", x509.HostnameError{Host: "h"}),
		x509.CertificateInvalidError{},
		errors.New("Get x: tls: failed to verify certificate: x509: certificate signed by unknown authority"),
	}
	for _, e := range yes {
		if !IsCertError(e) {
			t.Errorf("want cert error: %v", e)
		}
	}
	if IsCertError(nil) || IsCertError(errors.New("connection refused")) {
		t.Fatal("false positive")
	}
}
