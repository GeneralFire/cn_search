// Package ocr — HTTP-клиент сервиса распознавания номеров (каталог ocr/ в корне репозитория):
// POST /recognize, multipart-поле image → JSON [{plate, confidence, bbox}]; POST /anonymize → JPEG с размытыми лицами.
package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Timeout — сколько ждём ответа OCR (на CPU фото распознаётся за десятки–сотни миллисекунд).
const Timeout = 15 * time.Second

// maxAnonymized — предел размера ответа /anonymize.
const maxAnonymized = 32 << 20

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

// post отправляет фото multipart-полем image на path; 4xx возвращает как (resp.StatusCode, nil body).
func (c *Client) post(ctx context.Context, path string, photo []byte) (*http.Response, error) {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrRecognizerUnavailable, err)
	}
	return resp, nil
}

// Anonymize отправляет фото на /anonymize и возвращает JPEG с размытыми лицами. Любой сбой
// (сеть, таймаут, не 200, пустой ответ) — models.ErrRecognizerUnavailable: неразмытое фото не сохраняем.
func (c *Client) Anonymize(ctx context.Context, photo []byte) ([]byte, error) {
	resp, err := c.post(ctx, "/anonymize", photo)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: anonymize status %d", models.ErrRecognizerUnavailable, resp.StatusCode)
	}
	out, err := io.ReadAll(io.LimitReader(resp.Body, maxAnonymized))
	if err != nil || len(out) == 0 {
		return nil, fmt.Errorf("%w: anonymize bad response: %v", models.ErrRecognizerUnavailable, err)
	}
	return out, nil
}

// Recognize отправляет фото и возвращает номера как прочитала модель (Raw). Сеть, таймаут, 5xx —
// models.ErrRecognizerUnavailable; 4xx (не картинка) — пустой список.
func (c *Client) Recognize(ctx context.Context, photo []byte) ([]models.PlateCandidate, error) {
	resp, err := c.post(ctx, "/recognize", photo)
	if err != nil {
		return nil, err
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
