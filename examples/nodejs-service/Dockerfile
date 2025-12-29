FROM node:22-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --production

COPY . .

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app .

EXPOSE 8080
CMD ["node", "index.js"]
