# edgeball

自作CDN 学習用。

[cdn-up-and-running](https://github.com/leandromoreira/cdn-up-and-running) をベースに、Go・素のnginx・k6でCDNを段階的に組み立て、Prometheus/Grafanaで挙動を計測する。構成案は [docs/architecture.md](docs/architecture.md)。

## 必要なもの

- Docker(Compose v2)
- Go 1.25 以上
- golangci-lint v2(`brew install golangci-lint`)

k6はDockerイメージで動かすのでインストール不要。

## 使い方

```sh
make up      # 起動
make bench   # 負荷をかける(SCENARIO=loadtest/<name>.js で切り替え)
make down    # 停止
```

- Grafana: http://localhost:3000
- Prometheus: http://localhost:9090

## フェーズ

| フェーズ | 内容 | タグ | ノート |
|---|---|---|---|
| Phase 1 | オリジンと計測基盤 | `v1.0.0` | [phase1](docs/notes/phase1.md) |
| Phase 2 | エッジでのキャッシュ | `v2.0.0` | |
| Phase 3 | チューニング(cache lock、stale) | `v3.0.0` | |
| Phase 4 | 複数エッジと負荷分散 | `v4.0.0` | |
| Phase 5 | エッジをGoで自作 | `v5.0.0` | |
| Phase 6 | 発展(purge、shield など) | | |

各フェーズの状態は `git checkout vN.0.0` で戻って確認できる。
