package config

import (
	"testing"
	"time"
)

func TestContractConfigV7Defaults(t *testing.T) {
	c, err := Parse([]byte(v5Base))
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTPTimeout() != 30*time.Second || c.HTTPRetries() != 3 {
		t.Fatal(c.HTTPTimeout(), c.HTTPRetries())
	}
}

func TestContractConfigV7Values(t *testing.T) {
	c, err := Parse([]byte(v5Base + "http:\n  timeout: 5s\n  retries: 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTPTimeout() != 5*time.Second || c.HTTPRetries() != 0 {
		t.Fatal(c.HTTPTimeout(), c.HTTPRetries())
	}
}

func TestContractConfigV7Invalid(t *testing.T) {
	for _, y := range []string{"timeout: 500ms", "timeout: 6m", "retries: -1", "retries: 11", "timeout: abc"} {
		if _, err := Parse([]byte(v5Base + "http:\n  " + y + "\n")); err == nil {
			t.Errorf("%s: want error", y)
		}
	}
}
