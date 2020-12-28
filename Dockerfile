FROM golang:alpine3.10 as builder
ARG helm_version=v3.4.0
RUN apk update && apk add curl
RUN mkdir /build
WORKDIR /build
RUN curl -LO https://get.helm.sh/helm-${helm_version}-linux-amd64.tar.gz
RUN tar xzf helm*.tar.gz
RUN chmod a+x linux-amd64/helm
ADD src /build/src
WORKDIR /build/src
#RUN go build -mod=vendor -ldflags="-s -w" -o ../cleaner .
RUN go build -ldflags="-s -w" -o ../cleaner .
FROM alpine:3.10.3
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/cleaner /app/
COPY --from=builder /build/linux-amd64/helm /bin
WORKDIR /app
ENTRYPOINT ["/app/cleaner"]