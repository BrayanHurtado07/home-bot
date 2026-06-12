# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app as a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o home-bot cmd/bot/main.go

# Stage 2: Create a minimal image for running the bot
FROM alpine:latest  

# Install ca-certificates and tzdata for TLS requests to Telegram/Groq and correct timezones in scheduler
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/home-bot .

# Command to run the executable
CMD ["./home-bot"]
