# Build stage
FROM oven/bun:alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json bun.lock* ./
RUN bun install

# Copy source code
COPY . .

# Build the application
RUN bun run build

# Production stage - Chainguard Wolfi (secure, minimal CVEs)
FROM cgr.dev/chainguard/wolfi-base:latest

ARG CADDY_VERSION=""

# Install Caddy
RUN if [ -z "$CADDY_VERSION" ]; then \
      apk add --no-cache caddy; \
    else \
      apk add --no-cache caddy~${CADDY_VERSION}; \
    fi

# Copy Caddyfile
COPY Caddyfile /etc/caddy/Caddyfile

# Copy built assets
COPY --from=builder /app/dist /srv

# Non-root user
USER nonroot

# Expose port
EXPOSE 3000

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile"]
