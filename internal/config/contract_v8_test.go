package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContractConfigV8CAFile(t *testing.T) {
	c, err := Parse([]byte(v5Base))
	if err != nil || c.HTTPCAFile() != "" {
		t.Fatal(err, c.HTTPCAFile())
	}
	c, err = Parse([]byte(v5Base + "http:\n  ca_file: /etc/ca.pem\n"))
	if err != nil || c.HTTPCAFile() != "/etc/ca.pem" {
		t.Fatal(err, c.HTTPCAFile())
	}
	t.Setenv("HOME", "/home/x")
	c, err = Parse([]byte(v5Base + "http:\n  ca_file: ~/certs/ca.crt\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/home/x", "certs/ca.crt"); c.HTTPCAFile() != want {
		t.Fatal(c.HTTPCAFile())
	}
}

func TestContractConfigV8Example(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "config.example.yml"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLANE_API_KEY", "k")
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
}
