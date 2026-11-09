FROM golang:1.27-alpine AS build
# Модули только из публичного прокси, с проверкой контрольных сумм.
ENV GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/be ./cmd

FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/be /usr/local/bin/be
ENV PORT=8080
EXPOSE 8080 8082
HEALTHCHECK --interval=5s --timeout=2s --retries=10 \
  CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["be"]
