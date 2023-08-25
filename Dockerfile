# TODO is there fresher base
FROM golang:1.16-buster AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY *.go ./
RUN go build -o /gothchat

# TODO Alpine linux
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=builder /gothchat /gothchat

EXPOSE 8080

ENTRYPOINT ["/gothchat"]
