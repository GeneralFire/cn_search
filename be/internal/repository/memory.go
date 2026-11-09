package repository

import (
	"context"
	"slices"
	"strings"
	"unicode"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// MemoryStore — хранилище в памяти (для тестов). Неизменяемо после создания,
// поэтому безопасно для конкурентного чтения без блокировок.
type MemoryStore struct {
	items []models.PlateSighting
	byID  map[int]int // id → позиция в items
}

var _ Repository = (*MemoryStore)(nil)

// NewMemoryStore строит хранилище; записи упорядочиваются от новых к старым.
func NewMemoryStore(all []models.PlateSighting) *MemoryStore {
	items := slices.Clone(all)
	slices.SortStableFunc(items, func(a, b models.PlateSighting) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return a.ID - b.ID
	})
	byID := make(map[int]int, len(items))
	for i, c := range items {
		byID[c.ID] = i
	}
	return &MemoryStore{items: items, byID: byID}
}

// Search находит записи по частям номера и возвращает страницу.
func (s *MemoryStore) Search(_ context.Context, q Query) (Page, error) {
	words := tsWords(q.Plate)
	page := Page{Items: []models.PlateSighting{}, Offset: q.Offset, Limit: q.Limit}
	for _, c := range s.items {
		if !matchWords(c.Plate, words) {
			continue
		}
		if page.Total >= q.Offset && len(page.Items) < q.Limit {
			page.Items = append(page.Items, c)
		}
		page.Total++
	}
	if next := q.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// Get возвращает запись по id.
func (s *MemoryStore) Get(_ context.Context, id int) (models.PlateSighting, error) {
	i, ok := s.byID[id]
	if !ok {
		return models.PlateSighting{}, ErrNotFound
	}
	return s.items[i], nil
}

// tsWords — грубая копия парсера 'simple': слова из букв и цифр в нижнем регистре.
func tsWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// matchWords повторяет to_tsvector('simple', plate) @@ plainto_tsquery('simple', q):
// каждое слово запроса совпадает с каким-то словом номера.
func matchWords(plate string, query []string) bool {
	words := tsWords(plate)
	for _, q := range query {
		if !slices.Contains(words, q) {
			return false
		}
	}
	return true
}
