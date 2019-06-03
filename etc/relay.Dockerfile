FROM golang:1.12 AS build

COPY . /relay_server

WORKDIR /relay_server

RUN mkdir -p build && go build -o build/relay examples/relay/*.go

FROM ubuntu:18.04

COPY --from=build /relay_server/build/relay /relay

ENTRYPOINT [ "/relay" ]
