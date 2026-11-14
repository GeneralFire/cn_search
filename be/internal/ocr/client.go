// Package ocr — HTTP-клиент сервиса распознавания номеров (каталог ocr/ в корне репозитория):
// POST /recognize, multipart-поле image → JSON [{plate, confidence, bbox}].
package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Timeout — сколько ждём ответа OCR (на CPU фото распознаётся за десятки–сотни миллисекунд).
const Timeout = 15 * time.Second

// Client — клиент OCR-сервиса.
type Client struct {
	url  string // напр. "http://ocr:8000"
	http *http.Client
}

func New(baseURL string) *Client {
	return &Client{url: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: Timeout}}
}

type candidate struct {
	Plate      string  `json:"plate"`
	Confidence float64 `json:"confidence"`
	BBox       []int   `json:"bbox"`
}

// Recognize отправляет фото и возвращает номера как прочитала модель (Raw). Сеть, таймаут, 5xx —
// models.ErrRecognizerUnavailable; 4xx (не картинка) — пустой список.
func (c *Client) Recognize(ctx context.Context, photo []byte) ([]models.PlateCandidate, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("image", "photo.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(photo); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/recognize", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrRecognizerUnavailable, err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode >= 500:
		return nil, fmt.Errorf("%w: status %d", models.ErrRecognizerUnavailable, resp.StatusCode)
	case resp.StatusCode >= 400:
		return nil, nil
	}
	var raw []candidate
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: bad response: %v", models.ErrRecognizerUnavailable, err)
	}
	out := make([]models.PlateCandidate, 0, len(raw))
	for _, r := range raw {
		out = append(out, models.PlateCandidate{Raw: r.Plate, Confidence: r.Confidence, BBox: r.BBox})
	}
	return out, nil
}
