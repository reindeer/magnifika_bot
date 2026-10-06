FROM            golang:1.27-alpine AS build
WORKDIR         /app
RUN             apk add make
COPY            go.mod go.sum ./
RUN             go mod download
COPY            . .
RUN             make build

FROM            alpine:3.22 AS app
RUN             apk add --no-cache ca-certificates tzdata
COPY            --from=build /app/bin/* /bin/
WORKDIR         /app
ENTRYPOINT      ["app"]
