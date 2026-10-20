package app

import "testing"

// Find при недоступной сети: проект и задача — из кэша, приложение помечено офлайн.
func TestFindOfflineFromCache(t *testing.T) {
	e := newEnv(t)
	flWarm(t, e)
	if _, err := e.app.Projects(ctx()); err != nil {
		t.Fatal(err)
	}
	e.rebuild() // новый процесс, тот же диск
	e.failAll(flNet)
	p, it, err := e.app.Find(ctx(), "PROJ", 42)
	if err != nil || p.ID != "p1" || it.ID != "i42" {
		t.Fatalf("Find: %v %v %v", p, it, err)
	}
	if off, at := e.app.Offline(); !off || !at.Equal(t0) {
		t.Fatalf("Offline = %v %v", off, at)
	}
	// задачи нет в кэше — сетевая ошибка не скрывается
	if _, _, err := e.app.Find(ctx(), "PROJ", 99); err == nil {
		t.Fatal("ждали ошибку для задачи вне кэша")
	}
}
