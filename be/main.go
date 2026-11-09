// Заглушка бэкенда: бэкенд пока не разрабатывается.
// Отвечает только на GET /api/health, чтобы контейнер поднимался в docker-compose.
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("be stub: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
