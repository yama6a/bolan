# Build
FROM golang:1.18-alpine AS build

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

RUN apk --update --no-cache add make

WORKDIR /app
ADD .. /app

RUN make build

# Run
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=build /app/.build/.artifacts/crawler /crawler
COPY --from=build /app/.build/.artifacts/service /service
USER 65532:65532

ENTRYPOINT ["/service"]
