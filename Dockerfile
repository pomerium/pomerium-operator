FROM golang:1.13 as builder

ENV CGO_ENABLED=0
ENV GO111MODULE=on

COPY . /build

WORKDIR /build

RUN curl -sL https://taskfile.dev/install.sh | sh
RUN go mod download

RUN ./bin/task build

FROM gcr.io/distroless/static
COPY --from=builder /build/pomerium-operator /
ENTRYPOINT ["/pomerium-operator"]
