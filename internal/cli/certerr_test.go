package cli

import (
	"bytes"
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/config"
)

func certRun(t *testing.T, berr error) (int, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("api_key: k\nworkspace: ws\ndefault_project: PROJ\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	env := Env{Stdout: &out, Stderr: &errOut, Getenv: func(string) string { return "" },
		Getwd: func() (string, error) { return "/src/repo", nil },
		Build: func(*config.Config, string) (*app.App, error) { return nil, berr }}
	return Run(t.Context(), []string{"--config", path, "ls"}, env), errOut.String()
}

func TestContractCLICertHint(t *testing.T) {
	code, stderr := certRun(t, x509.UnknownAuthorityError{})
	if code != 4 || !strings.Contains(stderr, "platui: "+app.CertHint) {
		t.Fatal(code, stderr)
	}
}

func TestContractCLINoCertHint(t *testing.T) {
	code, stderr := certRun(t, errors.New("dial tcp: connection refused"))
	if strings.Contains(stderr, app.CertHint) {
		t.Fatal(code, stderr)
	}
}
