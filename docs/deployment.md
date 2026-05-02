# Ligo Deployment Guide

This guide covers deploying Ligo applications to various platforms.

## Table of Contents

- [Build Configuration](#build-configuration)
- [Environment Variables](#environment-variables)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Cloud Platforms](#cloud-platforms)
- [Monitoring and Logging](#monitoring-and-logging)
- [Security](#security)
- [Health Checks](#health-checks)

## Build Configuration

### Build Flags

Use appropriate build flags for production:

```bash
# Optimized build
go build -ldflags="-s -w" -o app ./cmd/app

# With version info
VERSION=$(git describe --tags --always)
go build -ldflags="-s -w -X main.Version=$VERSION" -o app ./cmd/app
```

### Build Tags

Use build tags for platform-specific code:

```go
// +build !development

package main

// Production-only code
```

```bash
# Build without development tools
go build -tags=!development -o app ./cmd/app
```

## Environment Variables

### Configuration Structure

```go
type Config struct {
    // Server
    Port        int
    Host        string
    Environment string

    // Database
    DatabaseURL string

    // Redis
    RedisURL string

    // Security
    SecretKey string

    // Features
    EnableMetrics bool
    EnableTracing bool
}

func LoadConfig() *Config {
    return &Config{
        Port:         getEnv("PORT", 8080),
        Host:         getEnv("HOST", "0.0.0.0"),
        Environment:  getEnv("ENV", "development"),
        DatabaseURL:  getEnv("DATABASE_URL", ""),
        RedisURL:     getEnv("REDIS_URL", ""),
        SecretKey:    getEnv("SECRET_KEY", ""),
        EnableMetrics: getEnvBool("ENABLE_METRICS", false),
        EnableTracing: getEnvBool("ENABLE_TRACING", false),
    }
}

func getEnv(key string, defaultValue any) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fmt.Sprint(defaultValue)
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        return value == "true" || value == "1"
    }
    return defaultValue
}
```

### .env File (Development Only)

```bash
# .env
PORT=8080
ENV=development
DATABASE_URL=postgres://localhost:5432/mydb
SECRET_KEY=dev-secret-key
```

```go
// Load .env in development
import "github.com/joho/godotenv"

func main() {
    if os.Getenv("ENV") != "production" {
        godotenv.Load()
    }
    // ...
}
```

## Docker Deployment

### Dockerfile

```dockerfile
# Multi-stage build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/app

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/app .

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./app"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - ENV=production
      - DATABASE_URL=postgres://postgres:password@db:5432/mydb
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s

  db:
    image: postgres:16-alpine
    environment:
      - POSTGRES_DB=mydb
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    restart: unless-stopped

volumes:
  postgres_data:
```

### Build and Run

```bash
# Build image
docker build -t myapp:latest .

# Run container
docker run -p 8080:8080 \
    -e ENV=production \
    -e DATABASE_URL=postgres://... \
    myapp:latest

# Use docker compose
docker compose up -d
```

## Kubernetes Deployment

### Deployment YAML

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  labels:
    app: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: ENV
          value: "production"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: myapp-secrets
              key: database-url
        - name: SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: myapp-secrets
              key: secret-key
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
    name: http
  type: ClusterIP
---
apiVersion: v1
kind: Secret
metadata:
  name: myapp-secrets
type: Opaque
stringData:
  database-url: "postgres://..."
  secret-key: "your-secret-key"
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: myapp-config
data:
  ENV: "production"
  ENABLE_METRICS: "true"
  ENABLE_TRACING: "false"
```

### HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Cloud Platforms

### AWS ECS

```json
{
  "family": "myapp",
  "containerDefinitions": [
    {
      "name": "myapp",
      "image": "myapp:latest",
      "memory": 256,
      "cpu": 256,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "ENV",
          "value": "production"
        }
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:database-url"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/myapp",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "healthCheck": {
        "command": [
          "CMD-SHELL",
          "wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1"
        ],
        "interval": 30,
        "timeout": 5,
        "retries": 3
      }
    }
  ],
  "requiresCompatibilities": ["FARGATE"],
  "networkMode": "awsvpc",
  "cpu": "256",
  "memory": "512"
}
```

### Google Cloud Run

```dockerfile
# Use distroless for minimal image
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o app ./cmd/app

FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
```

```bash
# Deploy to Cloud Run
gcloud run deploy myapp \
    --image gcr.io/PROJECT_ID/myapp \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars ENV=production \
    --set-secrets DATABASE_URL=myapp-db-url:latest
```

### Azure Container Instances

```bash
# Create container group
az container create \
    --resource-group myResourceGroup \
    --name myapp \
    --image myapp:latest \
    --cpu 1 \
    --memory 1 \
    --ports 8080 \
    --environment-variables ENV=production \
    --secure-environment-variables DATABASE_URL=$DATABASE_URL
```

## Monitoring and Logging

### Structured Logging

```go
func main() {
    logger := ligo.NewLogger(
        ligo.WithLoggerJSON(),  // JSON for production
    )

    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithLogger(logger),
        ligo.WithAddr(":8080"),
    )
    app.Run()
}
```

### Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

func main() {
    // Metrics endpoint
    http.Handle("/metrics", promhttp.Handler())
    go http.ListenAndServe(":9090", nil)

    // Your app
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
    )
    app.Run()
}
```

### Distributed Tracing

```go
import "go.opentelemetry.io/otel"

func main() {
    // Initialize tracing
    tracerProvider := initTracing()
    defer tracerProvider.Shutdown(context.Background())

    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
    )
    app.Run()
}

func initTracing() *trace.TracerProvider {
    exporter, _ := jaeger.New(jaeger.WithCollectorEndpoint())
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
    )
    otel.SetTracerProvider(tp)
    return tp
}
```

## Security

### TLS/HTTPS

```go
import "golang.org/x/crypto/acme/autocert"

func main() {
    e := echo.New()

    // Auto-cert for Let's Encrypt
    certManager := &autocert.Manager{
        Prompt:     autocert.AcceptTOS,
        HostPolicy: autocert.HostWhitelist("example.com"),
        Cache:      autocert.DirCache("certs"),
    }

    server := &http.Server{
        Addr:    ":https",
        Handler: e,
        TLSConfig: &tls.Config{
            GetCertificate: certManager.GetCertificate,
            MinVersion:     tls.VersionTLS12,
        },
    }

    // Redirect HTTP to HTTPS
    go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

    // Start HTTPS server
    app := ligo.New(
        ligo.WithRouter(ligo.NewAdapter(e)),
    )
    app.RegisterModules()

    log.Fatal(server.ListenAndServeTLS("", ""))
}
```

### Security Headers Middleware

```go
func SecurityHeadersMiddleware(next ligo.HandlerFunc) ligo.HandlerFunc {
    return func(ctx ligo.Context) error {
        h := ctx.Response().Header()

        // Prevent clickjacking
        h.Set("X-Frame-Options", "DENY")

        // Prevent MIME sniffing
        h.Set("X-Content-Type-Options", "nosniff")

        // XSS protection
        h.Set("X-XSS-Protection", "1; mode=block")

        // Content Security Policy
        h.Set("Content-Security-Policy", "default-src 'self'")

        // Referrer Policy
        h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

        // HSTS
        h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

        return next(ctx)
    }
}
```

### Rate Limiting

```go
import "golang.org/x/time/rate"

func RateLimitMiddleware(requestsPerMinute int) ligo.Middleware {
    limiter := rate.NewLimiter(rate.Every(time.Minute/requestsPerMinute), 10)

    return func(next ligo.HandlerFunc) ligo.HandlerFunc {
        return func(ctx ligo.Context) error {
            if !limiter.Allow() {
                return ctx.TooManyRequests("rate limit exceeded")
            }
            return next(ctx)
        }
    }
}
```

## Health Checks

### Health Check Module

```go
package health

import (
    "database/sql"
    "github.com/linkeunid/ligo"
)

type Checker struct {
    db *sql.DB
}

func Module(db *sql.DB) ligo.Module {
    return ligo.NewModule("health",
        ligo.Providers(
            ligo.Factory[*Checker](func() *Checker {
                return &Checker{db: db}
            }),
        ),
        ligo.Controllers(func(c *Checker) ligo.Controller {
            return c
        }),
    )
}

func (c *Checker) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r)

    cr.GET("/health", c.Health).Handle()
    cr.GET("/ready", c.Ready).Handle()
}

func (c *Checker) Health(ctx ligo.Context) error {
    return ctx.OK(map[string]string{
        "status": "healthy",
    })
}

func (c *Checker) Ready(ctx ligo.Context) error {
    // Check database connection
    if err := c.db.Ping(); err != nil {
        return ctx.ServiceUnavailable("database not ready")
    }

    return ctx.OK(map[string]string{
        "status": "ready",
    })
}
```

### Graceful Shutdown

```go
func main() {
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
        ligo.WithGracefulShutdown(30 * time.Second),
        ligo.OnStop(func(ctx any) error {
            log.Println("Shutting down gracefully...")
            return nil
        }),
    )

    app.Run()
}
```

## Deployment Checklist

### Pre-Deployment
- [ ] All environment variables documented
- [ ] Secrets stored securely (not in code)
- [ ] Health check endpoints implemented
- [ ] Graceful shutdown configured
- [ ] Logging configured for production
- [ ] Error monitoring integrated

### Security
- [ ] TLS/HTTPS enabled
- [ ] Security headers configured
- [ ] Rate limiting implemented
- [ ] Input validation on all endpoints
- [ ] SQL injection prevention
- [ ] XSS prevention
- [ ] CSRF protection (if needed)

### Performance
- [ ] Connection pooling configured
- [ ] Response compression enabled
- [ ] Static asset caching
- [ ] Database indexes verified
- [ ] Memory limits set

### Monitoring
- [ ] Metrics endpoint exposed
- [ ] Logging to centralized service
- [ ] Error tracking (Sentry, etc.)
- [ ] Uptime monitoring
- [ ] Alert thresholds configured

### Scaling
- [ ] Horizontal scaling tested
- [ ] Load balancer configured
- [ ] Session state externalized (if used)
- [ ] Database connection pool sized appropriately
