package usecase

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// MinPlateConfidence — номер с меньшей уверенностью OCR считается нераспознанным.
const MinPlateConfidence = 0.5

// photoKeyPrefix — каталог загруженных фото в бакете (тестовые лежат в корне: car-N.jpg).
const photoKeyPrefix = "u/"

// Ошибки для пользователя.
const (
	errNotPhoto           = "файл не похож на фото: нужен JPEG или PNG"
	errPlateNotRecognized = "номер не распознан: сфотографируйте номер крупнее и чётче"
)

// Ingest — новая фиксация по фото: распознать номер, сохранить фото, записать номер и фиксацию.
type Ingest struct {
	tx     transactor
	repo   ingestRepository
	ocr    recognizer
	photos photoStorage

	now   func() time.Time
	newID func() string
}

func NewIngest(tx transactor, repo ingestRepository, ocr recognizer, photos photoStorage) *Ingest {
	return &Ingest{tx: tx, repo: repo, ocr: ocr, photos: photos, now: time.Now, newID: uuid.NewString}
}

// Ingest: фото декодируется (не фото — ValidationError), OCR находит номера, лучший по уверенности
// (не ниже MinPlateConfidence, иначе ValidationError) нормализуется по формату (classifyPlate).
// Фото (до 1600 px) и миниатюра (480 px) без EXIF — в S3, затем в одной транзакции номер
// находится или создаётся и создаётся фиксация. Время и координаты — из запроса, иначе из EXIF;
// время — иначе сейчас. Если транзакция не прошла, фото остаются в S3 (их подчистит будущая уборка).
func (u *Ingest) Ingest(ctx context.Context, o models.IngestOpts) (models.IngestResult, error) {
	img, meta, err := decodePhoto(o.Photo)
	if err != nil {
		return models.IngestResult{}, &models.ValidationError{Message: errNotPhoto}
	}
	cands, err := u.ocr.Recognize(ctx, o.Photo)
	if err != nil {
		return models.IngestResult{}, fmt.Errorf("recognize: %w", err)
	}
	for i := range cands {
		cands[i].Plate, cands[i].PlateType, cands[i].Region = classifyPlate(cands[i].Raw)
	}
	cands = slices.DeleteFunc(cands, func(c models.PlateCandidate) bool { return c.Plate == "" })
	slices.SortStableFunc(cands, func(a, b models.PlateCandidate) int { return cmp.Compare(b.Confidence, a.Confidence) })
	if len(cands) == 0 || cands[0].Confidence < MinPlateConfidence {
		return models.IngestResult{Candidates: cands}, &models.ValidationError{Message: errPlateNotRecognized}
	}
	best := cands[0]

	full, err := encodeJPEG(img, photoMaxWidth, draw.ApproxBiLinear)
	if err != nil {
		return models.IngestResult{}, fmt.Errorf("encode photo: %w", err)
	}
	thumb, err := encodeJPEG(img, thumbWidth, draw.CatmullRom)
	if err != nil {
		return models.IngestResult{}, fmt.Errorf("encode thumb: %w", err)
	}
	id := u.newID()
	so := models.CreateSightingOpts{
		PhotoKey: photoKeyPrefix + id + ".jpg",
		ThumbKey: photoKeyPrefix + id + "_thumb.jpg",
		Address:  strings.TrimSpace(o.Address),
	}
	so.AddressLocale = addressLocale(so.Address)
	so.CreatedAt = u.now()
	if t := cmp.Or(o.TakenAt, meta.TakenAt); t != nil && t.Before(so.CreatedAt) {
		so.CreatedAt = *t
	}
	so.Lat, so.Lon = o.Lat, o.Lon
	if so.Lat == nil || so.Lon == nil {
		so.Lat, so.Lon = meta.Lat, meta.Lon
	}
	if err := u.photos.Put(ctx, so.PhotoKey, full, "image/jpeg"); err != nil {
		return models.IngestResult{}, fmt.Errorf("put photo: %w", err)
	}
	if err := u.photos.Put(ctx, so.ThumbKey, thumb, "image/jpeg"); err != nil {
		return models.IngestResult{}, fmt.Errorf("put thumb: %w", err)
	}

	var s models.PlateSighting
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		so.PlateID, err = u.repo.UpsertPlate(ctx, tx, models.UpsertPlateOpts{Plate: best.Plate, Type: best.PlateType, Region: best.Region})
		if err != nil {
			return err
		}
		s, err = u.repo.CreateSighting(ctx, tx, so)
		return err
	})
	if err != nil {
		return models.IngestResult{}, err
	}
	return models.IngestResult{Sighting: s, Candidates: cands}, nil
}
