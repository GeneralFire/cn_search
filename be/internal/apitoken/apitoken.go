// Package apitoken — выпуск и проверка подписи API-токенов.
//
// Токен = "cst_" + base64url(payload) + "." + base64url(HMAC-SHA256(payload, ключ)),
// payload = {"tid": "<uuid>", "iat": <unix>}. В БД хранится только SignatureHash —
// hex(sha256(подпись)): по нему токен находится, но из него токен не восстановить.
package apitoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Prefix — префикс API-токена (пользовательские токены — "cs_…", их здесь не разбираем).
const Prefix = "cst_"

// ErrInvalid — токен не разобрался или подпись не сошлась.
var ErrInvalid = errors.New("api token: invalid")

var b64 = base64.RawURLEncoding

type payload struct {
	TID string `json:"tid"`
	IAT int64  `json:"iat"`
}

// Signer подписывает и проверяет токены ключом TOKEN_SIGNING_KEY.
type Signer struct {
	key []byte
}

func NewSigner(key string) *Signer {
	return &Signer{key: []byte(key)}
}

func (s *Signer) sign(p []byte) []byte {
	m := hmac.New(sha256.New, s.key)
	m.Write(p)
	return m.Sum(nil)
}

// SignatureHash — то, что хранится в api_tokens.signature_hash.
func SignatureHash(sig []byte) string {
	h := sha256.Sum256(sig)
	return hex.EncodeToString(h[:])
}

// Issue выпускает токен для id; возвращает токен и hash его подписи.
func (s *Signer) Issue(id string, iat time.Time) (token, sigHash string) {
	p, _ := json.Marshal(payload{TID: id, IAT: iat.Unix()})
	sig := s.sign(p)
	return Prefix + b64.EncodeToString(p) + "." + b64.EncodeToString(sig), SignatureHash(sig)
}

// Verify разбирает токен и сверяет подпись (постоянного времени); возвращает id токена и hash подписи.
func (s *Signer) Verify(token string) (id, sigHash string, err error) {
	rest, ok := strings.CutPrefix(token, Prefix)
	if !ok || len(s.key) == 0 {
		return "", "", ErrInvalid
	}
	ps, ss, ok := strings.Cut(rest, ".")
	if !ok {
		return "", "", ErrInvalid
	}
	p, err1 := b64.DecodeString(ps)
	sig, err2 := b64.DecodeString(ss)
	if err1 != nil || err2 != nil || !hmac.Equal(sig, s.sign(p)) {
		return "", "", ErrInvalid
	}
	var pl payload
	if json.Unmarshal(p, &pl) != nil || uuid.Validate(pl.TID) != nil {
		return "", "", ErrInvalid
	}
	return pl.TID, SignatureHash(sig), nil
}
