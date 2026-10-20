package app

import "slices"

func (a *App) bookmarkKey(projectID string) string { return a.key("bookmarks", projectID) }

// Bookmarks — ID закладок проекта в порядке добавления.
func (a *App) Bookmarks(projectID string) []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.bookmarksLocked(projectID)
}

func (a *App) bookmarksLocked(projectID string) []string {
	if b, ok := a.marks[projectID]; ok {
		return slices.Clone(b)
	}
	var ids []string
	a.load(a.bookmarkKey(projectID), &ids)
	return ids
}

// ToggleBookmark ставит/снимает закладку; on — закладка стоит после вызова.
// Сбой записи в кэш возвращается ошибкой, состояние в памяти сохраняется.
func (a *App) ToggleBookmark(projectID, itemID string) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := a.bookmarksLocked(projectID)
	on := !slices.Contains(ids, itemID)
	if on {
		ids = append(ids, itemID)
	} else {
		ids = slices.DeleteFunc(ids, func(s string) bool { return s == itemID })
	}
	if a.marks == nil {
		a.marks = map[string][]string{}
	}
	a.marks[projectID] = ids
	if a.d.Store != nil {
		if err := a.d.Store.Save(a.bookmarkKey(projectID), ids); err != nil {
			return on, err
		}
	}
	return on, nil
}
