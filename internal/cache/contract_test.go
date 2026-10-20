package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type item struct {
	A string
	N int
}

func TestContractCacheRoundTrip(t *testing.T) {
	s := NewFileStore(filepath.Join(t.TempDir(), "c"))
	var got item
	if _, ok, err := s.Load("k", &got); ok || err != nil {
		t.Fatalf("missing: ok=%v err=%v", ok, err)
	}
	before := time.Now().Add(-time.Second)
	if err := s.Save("k", item{"x", 3}); err != nil {
		t.Fatal(err)
	}
	saved, ok, err := s.Load("k", &got)
	if !ok || err != nil || got != (item{"x", 3}) || saved.Before(before) {
		t.Fatalf("got %v %v %v %v", got, saved, ok, err)
	}
}

func TestContractCacheBadFileAndVersion(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	var got item
	for name, body := range map[string]string{
		"bad": "{not json", "v2": `{"v":2,"saved":"2026-01-01T00:00:00Z","data":{}}`,
		"nodata": `{"v":1,"saved":"2026-01-01T00:00:00Z"}`,
		"type":   `{"v":1,"saved":"2026-01-01T00:00:00Z","data":[1]}`,
	} {
		os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600)
		if _, ok, err := s.Load(name, &got); ok || err != nil {
			t.Errorf("%s: ok=%v err=%v", name, ok, err)
		}
	}
}

func TestContractCacheKeysAndSubdirs(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	for _, k := range []string{"", "a b", "../x", "a//b", "/a", "a/", "a/./b", "ü", ".."} {
		if err := s.Save(k, 1); err == nil {
			t.Errorf("Save(%q) returned no error", k)
		}
		if _, _, err := s.Load(k, new(int)); err == nil {
			t.Errorf("Load(%q) returned no error", k)
		}
	}
	if err := s.Save("proj/a.b-c_1", 5); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "proj", "a.b-c_1.json")); err != nil {
		t.Fatal(err)
	}
}

func TestContractCachePermsAtomic(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(filepath.Join(dir, "d"))
	for i := 0; i < 3; i++ {
		if err := s.Save("a/k", i); err != nil {
			t.Fatal(err)
		}
	}
	fi, err := os.Stat(filepath.Join(dir, "d", "a", "k.json"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("%v %v", fi, err)
	}
	tmps, _ := filepath.Glob(filepath.Join(dir, "d", "a", "*.tmp"))
	if len(tmps) != 0 {
		t.Fatalf("tmp left behind: %v", tmps)
	}
	if err := s.Save("f", make(chan int)); err == nil {
		t.Error("expected a marshal error")
	}
}

func TestContractCacheSaveErrors(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	os.WriteFile(f, nil, 0o600)
	if err := NewFileStore(f).Save("k", 1); err == nil {
		t.Error("expected a MkdirAll error")
	}
	os.Mkdir(filepath.Join(dir, "k.json"), 0o700)
	os.WriteFile(filepath.Join(dir, "k.json", "x"), nil, 0o600)
	if err := NewFileStore(dir).Save("k", 1); err == nil {
		t.Error("expected a rename error")
	}
	if _, _, err := NewFileStore(dir).Load("k", new(int)); err == nil {
		t.Error("expected a directory read error")
	}
}

func TestContractCacheMem(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	m := NewMem(func() time.Time { return ts })
	var got item
	if _, ok, _ := m.Load("k", &got); ok {
		t.Fatal("ok")
	}
	if err := m.Save("k", item{"y", 1}); err != nil {
		t.Fatal(err)
	}
	saved, ok, err := m.Load("k", &got)
	if !ok || err != nil || got != (item{"y", 1}) || !saved.Equal(ts) {
		t.Fatal(got, saved, ok, err)
	}
	if m.Save("bad key", 1) == nil || m.Save("k", make(chan int)) == nil {
		t.Error("expected errors")
	}
	if _, _, err := m.Load("bad key", &got); err == nil {
		t.Error("key")
	}
	if NewMem(nil).Save("k", 1) != nil {
		t.Error("nil now")
	}
}
