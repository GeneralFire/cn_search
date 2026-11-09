package main

import (
	"context"
	"encoding/json"
	"net/http"

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
	Code    string `json:"code"`    // имя gRPC-кода: "InvalidArgument", "NotFound", …
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
	_ = json.NewEncoder(w).Encode(apiError{Code: st.Code().String(), Message: msg})
}
