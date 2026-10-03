package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

type item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// GeneratedAt はオリジンがレスポンスを作った時刻。エッジから返ってきた値と比べると、キャッシュから返ったかが分かる
	GeneratedAt time.Time `json:"generated_at"`
	Origin      string    `json:"origin"`
}

type itemsHandler struct {
	maxAge   int
	maxItems int
	latency  *latencyDist
	hostname string
	// sleep はテストで差し替える
	sleep func(time.Duration)
}

func newItemsHandler(maxAge, maxItems int, latency *latencyDist) *itemsHandler {
	hostname, _ := os.Hostname()
	return &itemsHandler{
		maxAge:   maxAge,
		maxItems: maxItems,
		latency:  latency,
		hostname: hostname,
		sleep:    time.Sleep,
	}
}

// ServeHTTP は GET /api/items/{id} を処理する。
//
// Cache-Control の max-age は既定で h.maxAge 秒。?max_age=N で上書きでき、N=0 なら no-store を返す。
func (h *itemsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 || id > h.maxItems {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}

	maxAge := h.maxAge
	if v := r.URL.Query().Get("max_age"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			http.Error(w, "max_age must be a non-negative integer", http.StatusBadRequest)
			return
		}
		maxAge = n
	}

	delay := h.latency.Next()
	h.sleep(delay)

	if maxAge == 0 {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	}
	w.Header().Set("Content-Type", "application/json")
	// 注入した遅延。エッジ経由のレスポンスと比べるときの手がかりにする
	w.Header().Set("X-Origin-Delay", delay.String())

	if err := json.NewEncoder(w).Encode(item{
		ID:          id,
		Name:        fmt.Sprintf("item-%d", id),
		GeneratedAt: time.Now().UTC(),
		Origin:      h.hostname,
	}); err != nil {
		slog.Warn("write response", "err", err)
	}
}
