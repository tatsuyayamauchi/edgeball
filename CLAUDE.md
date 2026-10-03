# edgeball

CDNの仕組みを自分で作って理解するための学習用リポジトリ。公開・運用はしない。
教材 [cdn-up-and-running](https://github.com/leandromoreira/cdn-up-and-running) をベースに、Luaを使わずGo・素のnginx・k6で組み立てる。

- 構成案とフェーズ計画: `docs/architecture.md`(作業前に該当フェーズを読むこと)
- フェーズごとの学習ノート: `docs/notes/phaseN.md`

## Claudeの関わり方(重要)

学習が目的なので、書く人を分ける。

| Claudeが実装してよい | ユーザーが書く(Claudeはヒント・レビュー・解説のみ) |
|---|---|
| Docker Compose、Prometheus・Grafana設定、ダッシュボード | nginxのキャッシュ設定(`proxy_cache*`、`proxy_cache_key`、`proxy_cache_lock`、`proxy_cache_use_stale` など) |
| Makefile、CI的な仕組み、k6シナリオの土台 | LBの振り分け設定(`hash ... consistent` など) |
| オリジン(Go)のAPI・遅延注入・メトリクス | Phase 5 のGoエッジの中核(キャッシュストア/LRU、鮮度判定、キャッシュキー、singleflight、stale-while-revalidate、条件付きリクエスト) |

- 右列の箇所は、ユーザーが明示的に「書いて」と頼まない限り完成コードを書かない。方針・関連ドキュメント・テストケース・TODOコメント付きの雛形までは出してよい
- レビューでは「どの商用CDN機能に当たるか」「RFC 9111のどこに当たるか」を添えて解説する
- 計測結果は数字で語る。推測で「速くなったはず」と書かない

## 開発ルール

- Gitは main に直接コミットする。フェーズ完了時に `vN.0.0` タグを打つ(Phase 1 = `v1.0.0`)
- Goは 1.27(`.tool-versions` で 1.27.1 に固定、Dockerビルドは `golang:1.27.1`)。Dockerイメージのタグは Docker Hub の最新安定版を固定で指定する(`latest` は使わない)
- Goはルートの単一モジュール(`github.com/tatsuyayamauchi/edgeball`)。標準ライブラリを優先し、外部依存は `docs/architecture.md` に挙がっているもの(`prometheus/client_golang`、`x/sync` など)に留める
- Luaは使わない
- Goファイルの編集後は hook で `gofmt` が自動で走る。コミット前に `make lint test` を通す
- ドキュメント・コメントは日本語

## コマンド

```sh
make up        # docker compose up -d --build
make down      # 停止(ボリュームは残す)
make clean     # 停止してボリュームも削除
make ps        # コンテナ一覧
make logs      # ログを追う(S=サービス名で絞り込み)
make reload    # 設定ファイルを読み直す(S=サービス名、既定は prometheus)
make test      # go test ./...
make lint      # golangci-lint run
make fmt       # gofmt
make bench     # k6を実行(SCENARIO=loadtest/<name>.js)
```

- Grafana: http://localhost:3000 (匿名で閲覧可、admin/admin)
- Prometheus: http://localhost:9090

## ディレクトリ

```
origin/         Go: オリジンのJSON API
edge-nginx/     Phase 1〜4: nginxエッジの設定
edge-go/        Phase 5: Go自作エッジ
lb/             LB設定
loadtest/       k6シナリオ
observability/  Prometheus設定、Grafanaのプロビジョニングとダッシュボード
docs/           構成案、議事録、学習ノート
```
