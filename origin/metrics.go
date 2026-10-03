package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// オリジンのメトリクス。Phase 2 以降は「エッジがどれだけオリジンへのリクエストを減らしたか」をこれで見る。
var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "origin_http_requests_total",
		Help: "オリジンが受けたHTTPリクエスト数",
	}, []string{"handler", "code"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "origin_http_request_duration_seconds",
		Help: "オリジンのレスポンス時間(注入した遅延を含む)",
		// 遅延注入の分布(数ms〜数秒)に合わせたバケット
		Buckets: []float64{.005, .01, .02, .03, .05, .075, .1, .2, .3, .5, .75, 1, 1.5, 2, 3, 5},
	}, []string{"handler"})

	requestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "origin_http_requests_in_flight",
		Help: "オリジンが処理中のリクエスト数",
	})
)

// instrument はハンドラにメトリクスを付ける。handler ラベルにはパスではなくルートのパターンを入れ、ラベルの種類が増えすぎないようにする。
func instrument(pattern string, h http.Handler) http.Handler {
	h = promhttp.InstrumentHandlerDuration(requestDuration.MustCurryWith(prometheus.Labels{"handler": pattern}), h)
	h = promhttp.InstrumentHandlerCounter(requestsTotal.MustCurryWith(prometheus.Labels{"handler": pattern}), h)
	return promhttp.InstrumentHandlerInFlight(requestsInFlight, h)
}
