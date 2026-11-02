# ---------- 前端构建 ----------
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---------- 后端构建 ----------
FROM golang:1.27-alpine AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /certpilot-server ./cmd/server

# ---------- 运行镜像 ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /certpilot-server ./
COPY --from=frontend /app/frontend/dist ./web/dist
EXPOSE 8080
ENTRYPOINT ["./certpilot-server"]
