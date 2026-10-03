.PHONY: up down clean ps logs test lint fmt bench

# make logs S=origin のようにサービスを絞り込める
S ?=
# make bench SCENARIO=loadtest/baseline.js
SCENARIO ?= loadtest/baseline.js

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

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

# k6はDockerで実行し、結果をPrometheusへremote writeする(Grafanaで見られる)
bench:
	docker compose run --rm k6 run -o experimental-prometheus-rw /$(SCENARIO)
