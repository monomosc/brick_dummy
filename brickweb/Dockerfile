ARG BUILDER_IMAGE
ARG RUNNER_IMAGE
FROM $BUILDER_IMAGE as builder
WORKDIR /brick
COPY . .
WORKDIR /brick/brickweb
RUN go build -mod=vendor

FROM $RUNNER_IMAGE
WORKDIR /brickweb
COPY --from=builder /brick/brickweb/brickweb .
ENTRYPOINT ["./brickweb", "--config", "/etc/brickweb/brickweb.json"]