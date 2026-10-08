package poker

import (
	"bytes"
	"embed"
	"net/http"
	"time"
)

//go:embed web/*
var pages embed.FS

func servePage(w http.ResponseWriter, r *http.Request) {
	file := ""
	switch r.URL.Path {
	case "/":
		file = "index.html"
	case "/app.js":
		file = "app.js"
	case "/style.css":
		file = "style.css"
	default:
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := pages.ReadFile("web/" + file)
	if err != nil {
		http.Error(w, "页面暂不可用", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; object-src 'none'; frame-ancestors 'none'")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, file, time.Time{}, bytes.NewReader(data))
}
