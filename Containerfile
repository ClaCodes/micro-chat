FROM golang:1.20.10-alpine3.17 AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

RUN apk add curl
RUN mkdir ./libs 
RUN curl https://unpkg.com/htmx.org@1.9.6/dist/htmx.min.js > ./libs/htmx.min.js
COPY ./views* ./views

COPY *.go ./
RUN go build -ldflags "-w" -o /microchat

FROM alpine:latest

WORKDIR /

COPY --from=builder /microchat /microchat

EXPOSE 8080

ENTRYPOINT ["/microchat"]
