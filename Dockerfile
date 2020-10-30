FROM golang:alpine3.10 as builder
ARG helm_version=v3.4.0
RUN apk update && apk add curl
RUN mkdir /build
ADD . /build/src
WORKDIR /build/src
RUN curl -LO https://get.helm.sh/helm-${helm_version}-linux-amd64.tar.gz
RUN go build -mod vendor -o ../cleaner .
RUN tar xzf helm*.tar.gz
RUN chmod a+x linux-amd64/helm
FROM alpine:3.10.3
COPY --from=builder /build/cleaner /app/
COPY --from=builder /build/src/linux-amd64/helm /bin
WORKDIR /app
ENTRYPOINT ["/app/cleaner"]