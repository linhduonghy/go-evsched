FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

ARG TARGET_OS=linux
ARG TARGET_ARCH=amd64
ARG OUTPUT_DIR=bin
ARG APP_NAME=evsched

COPY --chown=appuser:appgroup ${OUTPUT_DIR}/${TARGET_OS}-${TARGET_ARCH}/${APP_NAME} /app/evsched
COPY --chown=appuser:appgroup default.yaml /app/default.yaml

USER appuser

EXPOSE 9020

ENTRYPOINT ["/app/evsched"]