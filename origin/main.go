// origin は edgeball のオリジンサーバー。JSON API に Cache-Control と遅延注入を付けて返す。
//
// 環境変数:
//
//	ADDR       待ち受けアドレス(既定 :8080)
//	MAX_AGE    Cache-Control の max-age 秒(既定 60)
//	MAX_ITEMS  存在する item の数。これを超える id は 404(既定 10000)
//	LATENCY    遅延の分布。"<percentile>:<duration>" のカンマ区切り。"off" で無効(既定 "50:20ms,90:50ms,95:100ms,99:1s")
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := run(); err != nil {
		slog.Error("origin stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	addr := envOr("ADDR", ":8080")
	maxAge, err := envInt("MAX_AGE", 60)
	if err != nil {
		return err
	}
	maxItems, err := envInt("MAX_ITEMS", 10000)
	if err != nil {
		return err
	}
	latency, err := parseLatency(envOr("LATENCY", "50:20ms,90:50ms,95:100ms,99:1s"))
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	const itemsPattern = "GET /api/items/{id}"
	mux.Handle(itemsPattern, instrument(itemsPattern, newItemsHandler(maxAge, maxItems, latency)))
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("origin listening", "addr", addr, "max_age", maxAge, "max_items", maxItems, "latency", latency.points)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s=%q: must be a non-negative integer", key, v)
	}
	return n, nil
}
