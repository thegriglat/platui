// Package search реализует мгновенный нечёткий поиск по заголовкам задач
// (контракт C9 v4). Пакет листовой: документы собирает вызывающий код.
// Сопоставление — алгоритм fzf (github.com/junegunn/fzf/src/algo), свой не пишем.
package search

// MaxQuery — максимальная длина запроса в рунах; Search и ParseKey обрезают запрос.
const MaxQuery = 32

// Doc — документ для индексации.
type Doc struct {
	ID    string // ID задачи (UUID)
	Key   string // "PROJ-42"
	Title string
}

// Hit — найденный документ.
type Hit struct {
	ID    string
	Score int
	Pos   []int // позиции совпавших рун в Title, по возрастанию, без повторов
}
