package implementation

import (
	"github.com/GeneralFire/cn_search/be/internal/repository"
)

func newSvc() *CarSearch {
	all := repository.Generate(repository.DefaultGenerateOptions(func(id int) string { return "car-1.png" }))
	return NewCarSearch(repository.NewMemoryStore(all), "/s3/cars/")
}
