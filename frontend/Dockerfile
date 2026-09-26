# Stage 1: Build frontend with Node and pnpm
FROM node:22-alpine AS builder

# Enable pnpm via Corepack
RUN corepack enable && corepack prepare pnpm@10 --activate

WORKDIR /app

# Copy dependency definitions
COPY package.json pnpm-lock.yaml* ./

# Install dependencies
RUN pnpm install --no-frozen-lockfile

# Copy source code and configuration files
COPY . .

# Environment variable for Vite build
ARG VITE_API_URL=/api
ENV VITE_API_URL=${VITE_API_URL}

# Build SPA distribution
RUN pnpm build

# Stage 2: Serve static files with Nginx
FROM nginx:1.27-alpine AS runner

# Remove default Nginx welcome page
RUN rm -rf /usr/share/nginx/html/*

# Copy custom Nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy compiled SPA bundle from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose HTTP port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80/healthz || exit 1

CMD ["nginx", "-g", "daemon off;"]
