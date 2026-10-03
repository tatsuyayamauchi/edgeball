# ビルドコンテキストはリポジトリのルート(go.mod が1つのため)
FROM golang:1.27.1 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY origin/ origin/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/origin ./origin

FROM scratch
COPY --from=build /out/origin /origin
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/origin"]
