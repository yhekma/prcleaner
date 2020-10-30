FROM golang:alpine3.10 as builder
RUN mkdir /build
ADD . /build/src
WORKDIR /build/src
RUN go build -mod vendor -o ../cleaner .
FROM alpine:3.10.3
COPY --from=builder /build/cleaner /app/
WORKDIR /app
ENTRYPOINT ["/app/cleaner"]