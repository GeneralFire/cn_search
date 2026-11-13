package apitoken

import (
	"strings"
	"testing"
	"time"
)

func TestIssueVerify(t *testing.T) {
	s := NewSigner("dev-key")
	const id = "11111111-2222-3333-4444-555555555555"
	tok, hash := s.Issue(id, time.Unix(1700000000, 0))
	if !strings.HasPrefix(tok, Prefix) || len(hash) != 64 {
		t.Fatalf("Issue: %q %q", tok, hash)
	}
	gotID, gotHash, err := s.Verify(tok)
	if err != nil || gotID != id || gotHash != hash {
		t.Fatalf("Verify: %q %q %v", gotID, gotHash, err)
	}

	// Чужой ключ, испорченная подпись, подменённый payload, не тот префикс — недействителен.
	other, _ := NewSigner("other").Issue(id, time.Unix(1700000000, 0))
	forged, _ := NewSigner("dev-key").Issue("99999999-2222-3333-4444-555555555555", time.Unix(1, 0))
	pl, _, _ := strings.Cut(forged, ".")
	_, sig, _ := strings.Cut(tok, ".")
	for name, bad := range map[string]string{
		"чужой ключ":   other,
		"подпись":      tok[:len(tok)-2] + "AA",
		"подмена tid":  pl + "." + sig,
		"префикс":      "cs_" + strings.TrimPrefix(tok, Prefix),
		"без точки":    strings.ReplaceAll(tok, ".", ""),
		"пусто":        "",
		"мусор base64": Prefix + "!!!.???",
	} {
		if _, _, err := s.Verify(bad); err != ErrInvalid {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Пустой ключ — проверка всегда отказывает.
	if _, _, err := NewSigner("").Verify(tok); err != ErrInvalid {
		t.Errorf("пустой ключ: %v", err)
	}
}
