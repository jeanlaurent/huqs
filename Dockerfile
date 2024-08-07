FROM golang:1.22.5-alpine3.20 AS builder
COPY . /app
WORKDIR /app
RUN go build -o /app/huqs

FROM alpine:3.20
# Fix openssl version for alpine 3.20
RUN apk --no-cache upgrade openssl && apk add openssl=3.3.1-r3
RUN adduser -D appuser
USER appuser
COPY ./static /app/static
COPY --from=builder /app/huqs /app/huqs
ENV OP_CONNECT_HOST="http://localhost:8080"
ENV OP_CONNECT_TOKEN=""
ENV PORT="8080"
EXPOSE 8088

CMD ["/app/huqs"]