package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/utrack/clay/v3/transport/httpruntime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

// configureHTTPRuntime настраивает JSON-слой ручек clay.
func configureHTTPRuntime() {
	// По умолчанию proto3 JSON не передаёт нулевые значения: при пустой выдаче
	// ответ был бы {} без total и items. EmitUnpopulated отдаёт их всегда
	// (необязательный next_offset по-прежнему пропадает, когда страниц больше нет).
	jsonpb := &runtime.JSONPb{
		MarshalOptions:   protojson.MarshalOptions{EmitUnpopulated: true},
		UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
	}
	httpruntime.OverrideMarshaler("application/json", httpruntime.MarshalerPbJSON{
		Marshaler:   jsonpb,
		Unmarshaler: jsonpb,
	})

	httpruntime.SetError = writeAPIError
}

type apiError struct {
	Code    string `json:"code"`    // машиночитаемый: "invalid_argument", "not_found", …
	Message string `json:"message"` // для пользователя
}

// writeAPIError отдаёт ошибку как {"code","message"} с HTTP-статусом по gRPC-коду —
// без служебного префикса "rpc error: code = … desc = …".
func writeAPIError(_ context.Context, _ *http.Request, w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		// Например, невалидный JSON в теле запроса.
		st = status.New(codes.InvalidArgument, err.Error())
	}
	msg := st.Message()
	if st.Code() == codes.Internal || st.Code() == codes.Unknown {
		msg = "внутренняя ошибка сервера"
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(runtime.HTTPStatusFromCode(st.Code()))
	_ = json.NewEncoder(w).Encode(apiError{Code: snakeCode(st.Code()), Message: msg})
}

// snakeCode: codes.InvalidArgument → "invalid_argument".
func snakeCode(c codes.Code) string {
	var b strings.Builder
	for i, r := range c.String() {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}
