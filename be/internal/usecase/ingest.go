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

// UploadQuotaWindow — окно суточной квоты загрузок (скользящее: последние 24 часа).
const UploadQuotaWindow = 24 * time.Hour

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
	anon   anonymizer
	photos photoStorage

	now   func() time.Time
	newID func() string
}

func NewIngest(tx transactor, repo ingestRepository, ocr recognizer, anon anonymizer, photos photoStorage) *Ingest {
	return &Ingest{tx: tx, repo: repo, ocr: ocr, anon: anon, photos: photos, now: time.Now, newID: uuid.NewString}
}

// Ingest: фото декодируется (не фото — ValidationError), OCR находит номера, лучший по уверенности
// (не ниже MinPlateConfidence, иначе ValidationError) нормализуется по формату (classifyPlate).
// После распознавания фото анонимизируется (лица размыты, Anonymize), и уже оно (до 1600 px) с миниатюрой (480 px) без EXIF — в S3, затем в одной транзакции номер
// находится или создаётся и создаётся фиксация. Время и координаты — из запроса, иначе из EXIF;
// время — иначе сейчас. Если транзакция не прошла, фото остаются в S3 (их подчистит будущая уборка).
//
// Квота: Free — не больше FreeUploadsPerDay загрузок за UploadQuotaWindow (*models.UploadQuotaError),
// Pro — без ограничений. Проверяется до OCR (не тратить распознавание) и ещё раз в транзакции записи
// под блокировкой пользователя (параллельные загрузки не превысят лимит).
func (u *Ingest) Ingest(ctx context.Context, o models.IngestOpts) (models.IngestResult, error) {
	left, err := u.UploadsLeft(ctx, o.User)
	if err != nil {
		return models.IngestResult{}, err
	}
	if left != nil && *left == 0 {
		return models.IngestResult{}, &models.UploadQuotaError{Limit: FreeUploadsPerDay}
	}
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

	// Приватность: лица размываются до сохранения; сервис недоступен — не сохраняем неразмытое фото.
	// OCR выше смотрел оригинал (номера на нём не размыты), в S3 уходит только обработанная копия.
	blurred, err := u.anon.Anonymize(ctx, o.Photo)
	if err != nil {
		return models.IngestResult{}, fmt.Errorf("anonymize: %w", err)
	}
	// У обработанного JPEG нет EXIF, ориентация уже применена сервисом; EXIF-метаданные берём из оригинала.
	img, _, err = decodePhoto(blurred)
	if err != nil {
		return models.IngestResult{}, fmt.Errorf("anonymize: %w: bad image: %v", models.ErrRecognizerUnavailable, err)
	}

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
		PhotoKey:   photoKeyPrefix + id + ".jpg",
		ThumbKey:   photoKeyPrefix + id + "_thumb.jpg",
		Address:    strings.TrimSpace(o.Address),
		UploadedBy: o.User.ID,
		UploadedAt: u.now(),
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
		if o.User.Plan != models.PlanPro {
			if err := u.repo.LockUser(ctx, tx, o.User.ID); err != nil {
				return err
			}
			n, err := u.repo.CountUploads(ctx, tx, o.User.ID, so.UploadedAt.Add(-UploadQuotaWindow))
			if err != nil {
				return err
			}
			if UploadQuotaExceeded(o.User, n) {
				return &models.UploadQuotaError{Limit: FreeUploadsPerDay}
			}
			rest := FreeUploadsPerDay - n - 1
			left = &rest
		}
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
	return models.IngestResult{Sighting: s, Candidates: cands, UploadsLeft: left}, nil
}

// UploadsLeft — сколько загрузок пользователь ещё может сделать за UploadQuotaWindow; nil — без ограничений (Pro).
func (u *Ingest) UploadsLeft(ctx context.Context, user models.User) (*int, error) {
	if user.Plan == models.PlanPro {
		return nil, nil
	}
	var n int
	err := u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) (err error) {
		n, err = u.repo.CountUploads(ctx, tx, user.ID, u.now().Add(-UploadQuotaWindow))
		return err
	})
	if err != nil {
		return nil, err
	}
	left := max(0, FreeUploadsPerDay-n)
	return &left, nil
}
