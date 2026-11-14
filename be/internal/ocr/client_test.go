package ocr

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

func TestRecognize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, _, err := r.FormFile("image")
		if r.URL.Path != "/recognize" || err != nil {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		f.Close()
		_, _ = w.Write([]byte(`[{"plate":"P466YK79","confidence":0.94,"bbox":[1,2,3,4]}]`))
	}))
	defer srv.Close()

	got, err := New(srv.URL+"/").Recognize(context.Background(), []byte("jpeg"))
	if err != nil || len(got) != 1 || got[0].Raw != "P466YK79" || got[0].Confidence != 0.94 || len(got[0].BBox) != 4 {
		t.Fatalf("Recognize: %+v %v", got, err)
	}
}

func TestRecognizeUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := New(srv.URL).Recognize(context.Background(), nil); !errors.Is(err, models.ErrRecognizerUnavailable) {
		t.Errorf("err = %v", err)
	}
	srv.Close()
	if _, err := New(srv.URL).Recognize(context.Background(), nil); !errors.Is(err, models.ErrRecognizerUnavailable) {
		t.Errorf("закрытый сервер: err = %v", err)
	}
}
