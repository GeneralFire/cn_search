package cars

import (
	"context"
	"errors"
	"slices"
	"strings"
)

// ErrNotFound — машины с таким id нет.
var ErrNotFound = errors.New("car not found")

// Query — параметры поиска. Условия объединяются через И; пустое поле не ограничивает.
type Query struct {
	Plate  string // подстрока госномера (регистр, пробелы, латиница — не важны)
	Key    string // подстрока ключа (регистр, пробелы — не важны)
	Offset int
	Limit  int
}

// Page — страница результатов поиска.
type Page struct {
	Items      []Car
	Total      int // всего найдено по запросу
	Offset     int
	Limit      int
	NextOffset *int // nil — дальше ничего нет
}

// Repository — хранилище машин: PostgresStore в работе, MemoryStore — в тестах.
type Repository interface {
	Search(ctx context.Context, q Query) (Page, error)
	Get(ctx context.Context, id int) (Car, error)
}

// MemoryStore — хранилище в памяти (для тестов). Неизменяемо после создания,
// поэтому безопасно для конкурентного чтения без блокировок.
type MemoryStore struct {
	items []indexed
	byID  map[int]int // id → позиция в items
}

type indexed struct {
	car   Car
	plate string // нормализованный номер
	key   string // нормализованный ключ
}

var _ Repository = (*MemoryStore)(nil)

// NewMemoryStore строит хранилище; записи упорядочиваются от новых к старым.
func NewMemoryStore(all []Car) *MemoryStore {
	items := make([]indexed, len(all))
	for i, c := range all {
		items[i] = indexed{car: c, plate: NormalizePlate(c.Plate), key: NormalizeKey(c.Key)}
	}
	slices.SortStableFunc(items, func(a, b indexed) int {
		if c := strings.Compare(b.car.Date, a.car.Date); c != 0 {
			return c
		}
		return a.car.ID - b.car.ID
	})
	byID := make(map[int]int, len(items))
	for i, it := range items {
		byID[it.car.ID] = i
	}
	return &MemoryStore{items: items, byID: byID}
}

// Search находит машины, совпадающие и по номеру, и по ключу, и возвращает страницу.
func (s *MemoryStore) Search(_ context.Context, q Query) (Page, error) {
	plate, key := NormalizePlate(q.Plate), NormalizeKey(q.Key)

	page := Page{Items: []Car{}, Offset: q.Offset, Limit: q.Limit}
	for _, it := range s.items {
		if !strings.Contains(it.plate, plate) || !strings.Contains(it.key, key) {
			continue
		}
		if page.Total >= q.Offset && len(page.Items) < q.Limit {
			page.Items = append(page.Items, it.car)
		}
		page.Total++
	}
	if next := q.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// Get возвращает машину по id.
func (s *MemoryStore) Get(_ context.Context, id int) (Car, error) {
	i, ok := s.byID[id]
	if !ok {
		return Car{}, ErrNotFound
	}
	return s.items[i].car, nil
}
