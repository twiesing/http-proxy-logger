FROM golang:1.23-alpine as app-builder
WORKDIR /go/src/app
COPY . .
RUN go build -o app

FROM alpine
COPY --from=app-builder /go/src/app/app /app
ENTRYPOINT ["/app"]

