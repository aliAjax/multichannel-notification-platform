FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/notification-api ./cmd/notification-api
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/notification-worker ./cmd/notification-worker
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/notification-api /notification-api
COPY --from=build /out/notification-worker /notification-worker
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/notification-api"]

