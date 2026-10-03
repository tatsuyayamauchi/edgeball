package main

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"
)

// latencyPoint は「全リクエストの Percentile % がこの Delay 以下になる」という分布上の1点。
type latencyPoint struct {
	Percentile float64
	Delay      time.Duration
}

// latencyDist はパーセンタイル指定から遅延をサンプリングする。
// 教材(cdn-up-and-running)の ngx.sleep による遅延注入の置き換え。
//
// 点と点の間は線形補間する。先頭には暗黙に (0%, 0ms) を置き、最後の点より上は最後の値を使う。
// 例: "50:20ms,99:1s" なら p50≈20ms、p99≈1s になる。
type latencyDist struct {
	points []latencyPoint
}

// parseLatency は "50:20ms,90:50ms,99:1s" 形式を解釈する。空文字か "off" なら遅延なし。
func parseLatency(spec string) (*latencyDist, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "off" {
		return &latencyDist{}, nil
	}

	var points []latencyPoint
	for part := range strings.SplitSeq(spec, ",") {
		pStr, dStr, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok {
			return nil, fmt.Errorf("latency %q: want <percentile>:<duration>", part)
		}
		p, err := strconv.ParseFloat(pStr, 64)
		if err != nil || p <= 0 || p > 100 {
			return nil, fmt.Errorf("latency %q: percentile must be in (0, 100]", part)
		}
		d, err := time.ParseDuration(dStr)
		if err != nil || d < 0 {
			return nil, fmt.Errorf("latency %q: invalid duration", part)
		}
		points = append(points, latencyPoint{Percentile: p, Delay: d})
	}

	slices.SortFunc(points, func(a, b latencyPoint) int {
		switch {
		case a.Percentile < b.Percentile:
			return -1
		case a.Percentile > b.Percentile:
			return 1
		}
		return 0
	})
	for i := 1; i < len(points); i++ {
		if points[i].Percentile == points[i-1].Percentile {
			return nil, fmt.Errorf("latency: duplicate percentile %v", points[i].Percentile)
		}
		if points[i].Delay < points[i-1].Delay {
			return nil, fmt.Errorf("latency: delay must not decrease as percentile increases (p%v)", points[i].Percentile)
		}
	}
	return &latencyDist{points: points}, nil
}

// sample は u ∈ [0, 100) に対応する遅延を返す。
func (d *latencyDist) sample(u float64) time.Duration {
	prev := latencyPoint{}
	for _, p := range d.points {
		if u <= p.Percentile {
			ratio := (u - prev.Percentile) / (p.Percentile - prev.Percentile)
			return prev.Delay + time.Duration(ratio*float64(p.Delay-prev.Delay))
		}
		prev = p
	}
	return prev.Delay
}

// Next は分布に従ってランダムな遅延を返す。
func (d *latencyDist) Next() time.Duration {
	return d.sample(rand.Float64() * 100)
}
