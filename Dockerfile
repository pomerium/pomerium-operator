FROM golang:1.17 as builder

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN mkdir /build
WORKDIR /build

COPY go.mod go.sum /build/
RUN go mod download

COPY . /build

RUN curl -sL https://taskfile.dev/install.sh | sh

RUN ./bin/task build

FROM gcr.io/distroless/static
COPY --from=builder /build/pomerium-operator /
ENTRYPOINT ["/pomerium-operator"]
