FROM golang@sha256:ef5b4be1f94b36c90385abd9b6b4f201723ae28e71acacb76d00687333c17282 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY ./internal/ ./internal/
COPY ./cmd/ ./cmd/

FROM builder AS soundoff-controller-builder
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/soundoff-controller ./cmd/bot

FROM builder AS soundoff-worker-builder
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/soundoff-worker ./cmd/worker

FROM alpine@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1 AS soundoff-controller
COPY --from=soundoff-controller-builder /app/soundoff-controller /app/soundoff-controller
WORKDIR /app
ENTRYPOINT ["/app/soundoff-controller"]

FROM alpine@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1 AS soundoff-worker
COPY --from=soundoff-worker-builder /app/soundoff-worker /app/soundoff-worker
WORKDIR /app
ENTRYPOINT ["/app/soundoff-worker"]