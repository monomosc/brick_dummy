ARG BUILDER_IMAGE
ARG RUNNER_IMAGE
FROM $BUILDER_IMAGE as builder
WORKDIR /brick
COPY ./vendor .
COPY . .
WORKDIR /brick/brickvalidation
RUN go build -mod=vendor

FROM $RUNNER_IMAGE
WORKDIR /brrickvalidation
COPY --from=builder /brick/brickvalidation/brickvalidation .
ENTRYPOINT ["./brickvalidation", "--config", "/etc/brickvalidation/brickvalidation.json"]