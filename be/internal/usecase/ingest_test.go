package usecase

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	dbmocks "github.com/GeneralFire/cn_search/be/internal/db/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase/mocks"
)

// testJPEG — JPEG w×h без EXIF.
func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type ingestDeps struct {
	tx     *dbmocks.ExecutorMock
	repo   *mocks.IngestRepositoryMock
	ocr    *mocks.RecognizerMock
	photos *mocks.PhotoStorageMock
}

func newTestIngest(t *testing.T, withTx bool) (*Ingest, ingestDeps) {
	d := ingestDeps{dbmocks.NewExecutorMock(t), mocks.NewIngestRepositoryMock(t), mocks.NewRecognizerMock(t), mocks.NewPhotoStorageMock(t)}
	tr := mocks.NewTransactorMock(t)
	if withTx {
		tr = writeTx(t, d.tx)
	}
	u := NewIngest(tr, d.repo, d.ocr, d.photos)
	u.now = func() time.Time { return time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC) }
	u.newID = func() string { return "id1" }
	return u, d
}

func TestIngest(t *testing.T) {
	u, d := newTestIngest(t, true)
	photo := testJPEG(t, 2000, 1000)
	d.ocr.RecognizeMock.Expect(minimock.AnyContext, photo).Return([]models.PlateCandidate{
		{Raw: "155880", Confidence: 0.39},
		{Raw: "H311YO797", Confidence: 0.98, BBox: []int{1, 2, 3, 4}},
	}, nil)
	sizes := map[string]int{}
	d.photos.PutMock.Set(func(_ context.Context, key string, data []byte, ct string) error {
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil || ct != "image/jpeg" {
			t.Errorf("%s: %v %s", key, err, ct)
		}
		sizes[key] = cfg.Width
		return nil
	})
	d.repo.UpsertPlateMock.Expect(minimock.AnyContext, d.tx, models.UpsertPlateOpts{Plate: "Н311УО 797", Type: "ru"}).Return("p1", nil)
	lat, lon := 55.7, 37.6
	taken := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	d.repo.CreateSightingMock.Expect(minimock.AnyContext, d.tx, models.CreateSightingOpts{
		PlateID: "p1", Address: "Москва, Тверская 1", AddressLocale: "ru",
		PhotoKey: "u/id1.jpg", ThumbKey: "u/id1_thumb.jpg", CreatedAt: taken, Lat: &lat, Lon: &lon,
	}).Return(models.PlateSighting{ID: "s1", Plate: "Н311УО 797"}, nil)

	res, err := u.Ingest(context.Background(), models.IngestOpts{Photo: photo, TakenAt: &taken, Lat: &lat, Lon: &lon, Address: " Москва, Тверская 1 "})
	if err != nil {
		t.Fatal(err)
	}
	if res.Sighting.ID != "s1" || len(res.Candidates) != 2 || res.Candidates[0].Plate != "Н311УО 797" || res.Candidates[1].Raw != "155880" {
		t.Errorf("результат: %+v", res)
	}
	if sizes["u/id1.jpg"] != photoMaxWidth || sizes["u/id1_thumb.jpg"] != thumbWidth {
		t.Errorf("размеры: %v", sizes)
	}
}

// Время — сейчас, если его нет ни в запросе, ни в EXIF; время из будущего — тоже сейчас.
func TestIngestDefaults(t *testing.T) {
	u, d := newTestIngest(t, true)
	photo := testJPEG(t, 100, 50)
	future := u.now().Add(time.Hour)
	d.ocr.RecognizeMock.Return([]models.PlateCandidate{{Raw: "7ABC123", Confidence: 0.9}}, nil)
	d.photos.PutMock.Return(nil)
	d.repo.UpsertPlateMock.Expect(minimock.AnyContext, d.tx, models.UpsertPlateOpts{Plate: "7ABC123", Type: "us"}).Return("p1", nil)
	d.repo.CreateSightingMock.Expect(minimock.AnyContext, d.tx, models.CreateSightingOpts{
		PlateID: "p1", AddressLocale: "ru", PhotoKey: "u/id1.jpg", ThumbKey: "u/id1_thumb.jpg", CreatedAt: u.now(),
	}).Return(models.PlateSighting{ID: "s1"}, nil)

	if _, err := u.Ingest(context.Background(), models.IngestOpts{Photo: photo, TakenAt: &future}); err != nil {
		t.Fatal(err)
	}
}

func TestIngestErrors(t *testing.T) {
	var invalid *models.ValidationError

	u, _ := newTestIngest(t, false)
	if _, err := u.Ingest(context.Background(), models.IngestOpts{Photo: []byte("not an image")}); !errors.As(err, &invalid) || invalid.Message != errNotPhoto {
		t.Errorf("не фото: %v", err)
	}

	u, d := newTestIngest(t, false)
	d.ocr.RecognizeMock.Return([]models.PlateCandidate{{Raw: "P466YK79", Confidence: MinPlateConfidence - 0.01}}, nil)
	res, err := u.Ingest(context.Background(), models.IngestOpts{Photo: testJPEG(t, 10, 10)})
	if !errors.As(err, &invalid) || invalid.Message != errPlateNotRecognized || len(res.Candidates) != 1 {
		t.Errorf("ниже порога: %+v %v", res, err)
	}

	u, d = newTestIngest(t, false)
	d.ocr.RecognizeMock.Return(nil, models.ErrRecognizerUnavailable)
	if _, err := u.Ingest(context.Background(), models.IngestOpts{Photo: testJPEG(t, 10, 10)}); !errors.Is(err, models.ErrRecognizerUnavailable) {
		t.Errorf("OCR недоступен: %v", err)
	}
}

func TestClassifyPlate(t *testing.T) {
	for raw, want := range map[string][3]string{
		"P466YK79":   {"Р466УК 79", "ru", ""},
		"h311yo797":  {"Н311УО 797", "ru", ""},
		"O123AB":     {"О123АВ", "ru", ""},
		"0123A877":   {"", "", ""}, // буква в позиции цифры серии — не РФ
		"A1O3BC77":   {"А103ВС 77", "ru", ""},
		"7ABC123":    {"7ABC123", "us", ""},
		"AB123CD":    {"AB 123 CD", "eu", ""},
		"1234BCD":    {"1234 BCD", "eu", "E"},
		"MKD4419":    {"MKD4419", "eu", "D"},
		"京A12345":    {"京A12345", "", ""},
		"AA 1234-BC": {"AA1234BC", "", ""},
	} {
		p, typ, reg := classifyPlate(raw)
		if want[0] == "" { // не РФ, но и не другой формат — остаётся как есть
			want[0] = raw
		}
		if p != want[0] || typ != want[1] || reg != want[2] {
			t.Errorf("%q → %q %q %q, want %v", raw, p, typ, reg, want)
		}
	}
}

func TestAddressLocale(t *testing.T) {
	for addr, want := range map[string]string{
		"": "ru", "Москва, Тверская 1": "ru", "Київ, Хрещатик 1": "uk", "北京": "zh",
		"Berlin, Müllerstraße 5": "de", "Main St 5, Springfield": "en", "12": "ru",
	} {
		if got := addressLocale(addr); got != want {
			t.Errorf("%q → %q, want %q", addr, got, want)
		}
	}
}
