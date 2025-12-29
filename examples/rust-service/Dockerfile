FROM rust:1.75-alpine AS builder

RUN apk add --no-cache musl-dev

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src ./src

RUN cargo build --release

FROM alpine:3.19
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/target/release/rust-example /server

EXPOSE 8080
CMD ["/server"]
