# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
# CGO_ENABLED=0 for a statically linked binary, good for Alpine
# -ldflags="-w -s" to strip debug information and reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o transactions-api .

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/transactions-api .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./transactions-api"]
