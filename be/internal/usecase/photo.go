package usecase

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png" // PNG тоже принимаем
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
)

const (
	// photoMaxWidth — ширина полного фото в S3 (шире — уменьшается), как у тестовых car-N.jpg.
	photoMaxWidth = 1600
	// thumbWidth — ширина миниатюры для списков.
	thumbWidth  = 480
	jpegQuality = 85
)

// photoMeta — данные из EXIF фото; поля nil, если их нет.
type photoMeta struct {
	TakenAt  *time.Time
	Lat, Lon *float64
}

// decodePhoto декодирует JPEG/PNG, поворачивает по EXIF Orientation и читает EXIF: время съёмки и GPS.
func decodePhoto(data []byte) (image.Image, photoMeta, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, photoMeta{}, err
	}
	var meta photoMeta
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return img, meta, nil // EXIF нет или битый — не ошибка
	}
	if t, err := x.DateTime(); err == nil && !t.IsZero() {
		meta.TakenAt = &t
	}
	if lat, lon, err := x.LatLong(); err == nil && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 && (lat != 0 || lon != 0) {
		meta.Lat, meta.Lon = &lat, &lon
	}
	if tag, err := x.Get(exif.Orientation); err == nil {
		if o, err := tag.Int(0); err == nil {
			img = orient(img, o)
		}
	}
	return img, meta, nil
}

// orient поворачивает картинку по EXIF Orientation: 3 — 180°, 6 — 90° по часовой, 8 — 90° против.
// Зеркальные варианты (2, 4, 5, 7) с камер почти не встречаются — оставляем как есть.
func orient(src image.Image, o int) image.Image {
	if o != 3 && o != 6 && o != 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if o != 3 {
		w, h = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.At(b.Min.X+x, b.Min.Y+y)
			switch o {
			case 3:
				dst.Set(w-1-x, h-1-y, c)
			case 6:
				dst.Set(w-1-y, x, c)
			case 8:
				dst.Set(y, h-1-x, c)
			}
		}
	}
	return dst
}

// encodeJPEG уменьшает картинку до ширины maxWidth (уже — без изменений) и кодирует в JPEG.
// EXIF в результат не попадает (в том числе GPS) — так в S3 не утекают метаданные телефона.
func encodeJPEG(src image.Image, maxWidth int, scaler draw.Scaler) ([]byte, error) {
	b := src.Bounds()
	if b.Dx() > maxWidth {
		h := max(1, b.Dy()*maxWidth/b.Dx())
		dst := image.NewRGBA(image.Rect(0, 0, maxWidth, h))
		scaler.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
		src = dst
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
