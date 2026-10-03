# edgeball(学習用の自作CDN) 構成案

- プロダクト名: edgeball(GitHubのリポジトリ名として使う。名前へのこだわりはなし)
- ステータス: 構成案まとめ済み。本リポジトリで実装を進める(原本は ai-company リポジトリの `products/edgeball/architecture.md`)
- 壁打ちの議事録: `docs/2026-10-03_構成検討.md`
- 最終更新: 2026-10-03

## 0. 目的とスコープ

### 目的

CDNの仕組みを、自分で作って動かして理解する。プロダクトとして公開・運用するのが目的ではない。

社長はAkamai・CloudFront・Google Cloud CDNを利用者として知っているので、各フェーズで「商用CDNのどの機能に当たるか」を対応づけ、使ったことのある機能の中身を確かめる形で進める。

### やること / やらないこと

| やること | やらないこと |
|---|---|
| キャッシュするリバースプロキシ、TTL・キャッシュキー、同時リクエストの集約、stale配信、複数エッジへの負荷分散 | 本番運用、SLA、マルチテナント |
| メトリクスでの計測(ヒット率・レイテンシ・オリジン負荷) | 実際のAnycast(BGP)運用 |
| エッジをGoで自作し、nginxと比較する | WAF・DDoS対策・画像変換などの付加機能 |

## 1. 参考教材

[leandromoreira/cdn-up-and-running](https://github.com/leandromoreira/cdn-up-and-running) をベースにする。nginx + Lua + Docker Compose + Prometheus/Grafana でCDNを段階的に組み立てる教材で、v1〜v4のバージョンごとに機能が増えていく。

### 教材からの変更点(Luaを使わない)

社長はLuaを書いたことがないため、教材でLuaを使っている箇所を以下のように置き換える。CDNの核心(`proxy_cache` などのキャッシュ設定)はnginxの設定ファイル側にあるので、置き換えても学べる内容は変わらない。

| 教材での箇所 | 教材の実装 | edgeballでの実装 |
|---|---|---|
| バックエンド(オリジン) | OpenResty + `content_by_lua_block` | Goの `net/http` |
| 遅いオリジンの再現 | `ngx.sleep()` でパーセンタイル分布の遅延を注入 | Goのハンドラ内で同じ分布の遅延を入れる |
| エッジのキャッシュキー | `set_by_lua_block` | 素のnginxの `proxy_cache_key`(nginx変数のみ) |
| 負荷テスト | wrk + Luaスクリプト | k6(JavaScriptでシナリオを書く。k6自体もGo製) |

## 2. 技術スタック

| 役割 | 採用技術 | 備考 |
|---|---|---|
| オリジン | Go(標準ライブラリ中心) | JSON API。Cache-Controlヘッダと遅延注入を持つ |
| エッジ(前半) | nginx(OpenRestyではなく素のnginx) | `proxy_cache` 系の設定で動作を学ぶ |
| エッジ(後半) | Go(自作) | `httputil.ReverseProxy` + 自前キャッシュ |
| ロードバランサ | nginx(`upstream` + `hash ... consistent`) | 後半でGo自作も任意で |
| 実行環境 | Docker Compose(ローカル) | お金がかからず手元で完結する |
| メトリクス | Prometheus + Grafana | Goは `prometheus/client_golang`、nginxはexporterを使う |
| 負荷テスト | k6 | ロングテールなアクセス分布(人気URLに偏る)を再現する |

### 実行環境をローカルにする理由

学習の主眼はキャッシュの挙動で、それはローカルでも完全に再現できる。地理的な距離による遅延は `tc netem` でコンテナ間に人工的な遅延を入れて再現する(Phase 6)。VPSを複数リージョンに置くのは、ローカルで一通り終わってから検討すればよい。

## 3. 全体アーキテクチャ

### 最終形(Phase 4 時点)

```
                    ┌──────────┐
  k6 (負荷) ──────▶ │   LB     │  nginx: upstream hash $request_uri consistent
                    └────┬─────┘
           ┌─────────────┼─────────────┐
           ▼             ▼             ▼
      ┌────────┐    ┌────────┐    ┌────────┐
      │ edge1  │    │ edge2  │    │ edge3  │   nginx: proxy_cache
      └───┬────┘    └───┬────┘    └───┬────┘
          └─────────────┼─────────────┘
                        ▼
                   ┌─────────┐
                   │ origin  │   Go: JSON API + Cache-Control + 遅延注入
                   └─────────┘

  Prometheus ◀── 各コンテナのメトリクス ──▶ Grafana
```

### 発展形(Phase 5〜6)

```
  k6 ──▶ LB ──▶ edge1..3 (Go自作) ──▶ shield (中間キャッシュ) ──▶ origin
                     ▲
                purge API
```

## 4. フェーズ計画

各フェーズは「作る → k6で負荷をかける → Grafanaで数字を見る」の順で進め、数字の変化から挙動を理解する。

### Phase 1: オリジンと計測基盤(教材 v1.x 相当)

- 作るもの
  - Goのオリジン: `/api/items/{id}` のようなJSON APIを返す。レスポンスに `Cache-Control: max-age=N` を付ける
  - パーセンタイル分布(例: p50=20ms、p99=1s)に従った遅延注入
  - nginxのエッジ(キャッシュなし、単なるリバースプロキシ)
  - Prometheus + Grafana、オリジンのリクエスト数・レイテンシのメトリクス
- 理解すること: オリジンに全リクエストが届く状態(ベースライン)の数字

### Phase 2: エッジでのキャッシュ(教材 v2.x 相当)

- 作るもの: nginxに `proxy_cache_path` / `proxy_cache` / `proxy_cache_key` を設定。`X-Cache-Status` ヘッダでHIT/MISSを返す
- 計測: ヒット率、オリジンへのリクエスト数がどれだけ減るか
- 理解すること: キャッシュキーの設計(クエリ文字列やヘッダを含めるか)、オリジンのCache-ControlとエッジのTTLの関係
- 商用CDNとの対応
  - CloudFront: キャッシュポリシー(キャッシュキー、最小/デフォルト/最大TTL)
  - Google Cloud CDN: キャッシュモード(`USE_ORIGIN_HEADERS` / `CACHE_ALL_STATIC` / `FORCE_CACHE_ALL`)、キャッシュキーポリシー
  - Akamai: Property Managerのキャッシュ設定、Cache IDの変更

### Phase 3: チューニング(教材 v3.x 相当)

- 作るもの
  - `proxy_cache_lock`: 同じURLへの同時MISSをオリジンへ1本にまとめる
  - `proxy_cache_use_stale` / `proxy_cache_background_update`: 期限切れでも古いキャッシュを返しつつ裏で更新する
  - タイムアウト設定(`proxy_connect_timeout` など)
  - k6でロングテールなアクセス分布を再現する
- 理解すること: 人気コンテンツの期限切れの瞬間にオリジンへアクセスが殺到する問題(thundering herd)と、その防ぎ方
- 商用CDNとの対応
  - 同時リクエストの集約: CloudFrontのrequest collapsing、Cloud CDNのrequest coalescing
  - stale配信: CloudFrontの `stale-while-revalidate` / `stale-if-error` 対応、Cloud CDNのserve-while-stale、AkamaiのServe Stale

### Phase 4: 複数エッジと負荷分散(教材 v4.x 相当)

- 作るもの: エッジを3台に増やし、前段にLBを置く。ラウンドロビンとconsistent hashingを切り替えて比較する
- 計測: 方式ごとのヒット率の違い、エッジを1台止めたときのヒット率の落ち方
- 理解すること
  - ラウンドロビンだと同じURLが各エッジに分散し、キャッシュが重複してヒット率が下がる
  - consistent hashingだとURLごとに担当エッジが決まりヒット率が上がる。代わりに人気URLを持つエッジに負荷が偏る(ホットスポット)
  - consistent hashingではエッジの増減時に担当が移るキーが一部だけで済む
- 商用CDNとの対応: PoP内でキャッシュサーバーへ振り分ける仕組みに当たる。Akamaiはconsistent hashingの研究(Karger et al., MIT)から生まれた会社として知られる

### Phase 5: エッジをGoで自作する(独自パート)

nginxが内部でやっていることを自分で実装し、Phase 2〜4と同じk6シナリオで数字を比較する。ここが一番理解が深まるパート。

| 機能 | 実装方針 |
|---|---|
| リバースプロキシ | `net/http/httputil.ReverseProxy` |
| キャッシュストア | 自前のLRU(メモリ)。最初はmap + 双方向リストで自作し、必要なら `hashicorp/golang-lru/v2` と比較 |
| 鮮度判定 | `Cache-Control`(`max-age` / `s-maxage` / `no-store` / `private`)と `Age` ヘッダを自前でパース |
| キャッシュキー | メソッド + ホスト + パス + 正規化したクエリ |
| 同時MISSの集約 | `golang.org/x/sync/singleflight`(nginxの `proxy_cache_lock` 相当) |
| stale-while-revalidate | 期限切れエントリを返しつつgoroutineで裏更新 |
| メトリクス | `prometheus/client_golang` でHIT/MISS/STALE、オリジン応答時間 |
| 条件付きリクエスト | `ETag` / `If-None-Match` による再検証(304) |

- 理解すること: キャッシュ判定のロジック(RFC 9111)、メモリ上限と追い出し、並行処理での整合性
- 任意: LBもGoで自作し、consistent hashing(ハッシュリング + 仮想ノード)を自分で実装する

### Phase 6: 発展(任意。興味のあるものだけ)

| テーマ | 内容 | 商用CDNとの対応 |
|---|---|---|
| purge API | URL指定・プレフィックス指定でキャッシュを消す | CloudFront Invalidation、Cloud CDNのキャッシュ無効化、Akamai Fast Purge |
| 階層キャッシュ | エッジとオリジンの間に中間キャッシュ(shield)を置く | CloudFront Regional Edge Cache / Origin Shield、Akamai Tiered Distribution |
| 地理的な振り分け | CoreDNSで問い合わせ元に応じて別のエッジを返す。`tc netem` で拠点間の遅延を再現 | CloudFront・AkamaiのDNSベース振り分け、Cloud CDNのAnycast |
| ディスクキャッシュ | メモリとディスクの2層 | nginxの `proxy_cache_path` の中身 |
| Rangeリクエスト | 大きいファイルの部分取得と部分キャッシュ | 動画配信 |
| TLS終端 | エッジで証明書を扱う | 各CDNの証明書管理 |

## 5. 実装リポジトリの構成案(`edgeball` リポジトリ)

```
edgeball/
├── docker-compose.yml
├── origin/            # Go: オリジンのJSON API
├── edge-nginx/        # Phase 1〜4: nginx.conf をフェーズごとに
├── edge-go/           # Phase 5: Go自作エッジ
├── lb/                # nginxのLB設定(任意でGo自作)
├── loadtest/          # k6シナリオ
├── observability/     # Prometheus設定、Grafanaダッシュボード
└── docs/              # フェーズごとの学習ノート(計測結果と気づき)
```

教材にならい、フェーズの区切りでGitタグ(`v1.0.0` など)を打つと、あとから各段階の状態に戻って見返せる。

## 6. 決定事項

| 項目 | 決定内容 | 理由 |
|---|---|---|
| 名前 | edgeball | リポジトリ名程度の用途なのでこだわらない |
| 参考教材 | cdn-up-and-running | 段階的に組み立てられ、計測まで含む |
| 言語 | Go | HTTPまわりの標準ライブラリが充実。CDN関連のOSSにもGo製が多い |
| Lua | 使わない | 社長が未経験。Luaの箇所は置き換えても学習内容は変わらない |
| 実行環境 | ローカルのDocker Compose | 費用ゼロ、手元で完結 |

## 7. 未決事項

- Phase 6 のどのテーマまでやるか(Phase 5 まで終えた時点で決める)
- ローカル以外(複数リージョンのVPS)に出すかどうか
