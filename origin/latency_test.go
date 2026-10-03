package main

import (
	"slices"
	"testing"
	"time"
)

func TestParseLatency(t *testing.T) {
	tests := []struct {
		spec    string
		want    []latencyPoint
		wantErr bool
	}{
		{spec: "", want: nil},
		{spec: "off", want: nil},
		{spec: "50:20ms,99:1s", want: []latencyPoint{{50, 20 * time.Millisecond}, {99, time.Second}}},
		// 順不同でもパーセンタイル順に並べ替える
		{spec: "99:1s, 50:20ms", want: []latencyPoint{{50, 20 * time.Millisecond}, {99, time.Second}}},
		{spec: "50", wantErr: true},
		{spec: "0:10ms", wantErr: true},
		{spec: "101:10ms", wantErr: true},
		{spec: "50:abc", wantErr: true},
		{spec: "50:10ms,50:20ms", wantErr: true},
		{spec: "50:1s,99:10ms", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			got, err := parseLatency(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got %v", got.points)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got.points, tt.want) {
				t.Errorf("got %v, want %v", got.points, tt.want)
			}
		})
	}
}

func TestLatencySample(t *testing.T) {
	d, err := parseLatency("50:20ms,90:100ms,99:1s")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		u    float64
		want time.Duration
	}{
		{0, 0},
		{25, 10 * time.Millisecond}, // (0,0)〜(50,20ms) の中間
		{50, 20 * time.Millisecond},
		{70, 60 * time.Millisecond}, // (50,20ms)〜(90,100ms) の中間
		{99, time.Second},
		{99.9, time.Second}, // 最後の点より上は最後の値
	}
	for _, tt := range tests {
		if got := d.sample(tt.u); got != tt.want {
			t.Errorf("sample(%v) = %v, want %v", tt.u, got, tt.want)
		}
	}
}

// 実際にサンプリングしたとき、指定した遅延以下に収まる割合がパーセンタイルに近いことを確かめる。
// (分位点の値そのものを比べると、分布の傾きが急な区間で揺れが大きく不安定になる)
func TestLatencyNextDistribution(t *testing.T) {
	d, err := parseLatency("50:20ms,99:1s")
	if err != nil {
		t.Fatal(err)
	}
	const n = 100000
	var le20ms, le1s int
	for range n {
		v := d.Next()
		if v <= 20*time.Millisecond {
			le20ms++
		}
		if v <= time.Second {
			le1s++
		}
	}

	check := func(name string, count int, wantPercent float64) {
		t.Helper()
		got := float64(count) / n * 100
		if got < wantPercent-1 || got > wantPercent+1 {
			t.Errorf("%s: %.2f%% of samples, want %.0f%% ±1%%", name, got, wantPercent)
		}
	}
	check("<= 20ms", le20ms, 50)
	check("<= 1s", le1s, 100)
}

func TestLatencyOff(t *testing.T) {
	d, err := parseLatency("off")
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Next(); got != 0 {
		t.Errorf("Next() = %v, want 0", got)
	}
}
