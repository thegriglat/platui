package tui

import (
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/thegriglat/platui/internal/app"
)

func TestContractTUICertHint(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	got := m.s.errText(fmt.Errorf("Get x: %w", x509.UnknownAuthorityError{}))
	if !strings.Contains(got, app.CertHint) {
		t.Fatal(got)
	}
}

func TestContractTUINoCertHint(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	if got := m.s.errText(errors.New("connection refused")); strings.Contains(got, app.CertHint) {
		t.Fatal(got)
	}
}
