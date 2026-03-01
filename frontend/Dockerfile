# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --legacy-peer-deps

# Copy source code
COPY . .

# Build arguments for environment variables
ARG VITE_API_URL
ARG VITE_CONTRACT_ADDRESS
ARG VITE_USDC_ADDRESS
ARG VITE_ADMIN_PIN
ARG VITE_ADMIN_TOKEN
ARG VITE_ETH_RPC_URL

# Set environment variables for build
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_CONTRACT_ADDRESS=$VITE_CONTRACT_ADDRESS
ENV VITE_USDC_ADDRESS=$VITE_USDC_ADDRESS
ENV VITE_ADMIN_PIN=$VITE_ADMIN_PIN
ENV VITE_ADMIN_TOKEN=$VITE_ADMIN_TOKEN
ENV VITE_ETH_RPC_URL=$VITE_ETH_RPC_URL

# Build the application
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy built files
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
