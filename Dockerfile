# Build Stage
FROM golang:1.21-alpine AS build
RUN apk add --no-cache git
# Set the working directory
WORKDIR /app

# Copy all files to the working directory
COPY . .

# Download all dependencies
RUN go mod download

# Build the Go application
RUN go build -o /app/main ./cmd/backend/main.go

# Runtime Stage
FROM alpine:latest AS runtime
RUN apk --no-cache add ca-certificates

# Copy the binary from the build stage to the runtime stage
COPY --from=build /app/main /app/main
WORKDIR /app
# Set the entry point to execute the binary directly
ENTRYPOINT /app/main
