.PHONY: up down clean ps logs reload test lint fmt bench

# make logs S=origin のようにサービスを絞り込める
S ?=
# make bench SCENARIO=loadtest/baseline.js BASE_URL=http://origin:8080
SCENARIO ?= loadtest/baseline.js
BASE_URL ?=

up:
	docker compose up -d --build

down:
	docker compose down

clean:
	docker compose down -v

ps:
	docker compose ps

logs:
	docker compose logs -f $(S)

# Prometheus・nginxなど設定ファイルをマウントしているコンテナは、設定を変えても up だけでは読み直さない
reload:
	docker compose restart $(or $(S),prometheus)

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

# k6はDockerで実行し、結果をPrometheusへremote writeする(Grafanaで見られる)
bench:
	docker compose run --rm k6 run -o experimental-prometheus-rw $(if $(BASE_URL),-e BASE_URL=$(BASE_URL)) /$(SCENARIO)
