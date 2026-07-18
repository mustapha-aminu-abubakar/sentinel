# syntax=docker/dockerfile:1.7
# Builds the Next.js dashboard image using standalone output.
# Requires `output: 'standalone'` in next.config.ts (enabled in this change).
# Consumes env vars (no baked-in secrets):
#   NEXT_PUBLIC_SENTINEL_API_URL (browser-side base URL for the sentinel-api)
#   PORT (defaults to 3000 inside the standalone server)

FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
USER node
EXPOSE 3000
CMD ["node", "server.js"]
