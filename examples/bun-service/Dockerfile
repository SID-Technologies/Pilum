FROM oven/bun:1-alpine AS builder

WORKDIR /app
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile --production

COPY . .

FROM oven/bun:1-alpine
WORKDIR /app
COPY --from=builder /app .

EXPOSE 8080
CMD ["bun", "run", "index.ts"]
