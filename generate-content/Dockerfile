FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum (if they exist)
COPY go.mod go.sum* ./

# Uncomment if you have go.mod and go.sum
# RUN go mod download

# Copy source code
COPY *.go ./

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o pack-files .

# Create final lightweight image
FROM alpine:latest

# Install dependencies
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/pack-files .

# Create directory for storing output files
RUN mkdir -p /data/output

# Set default environment variables
ENV INPUT_DIR=/data/input
ENV INCLUDE_EXTS=md,go,mbt
ENV EXCLUDE_EXTS=html,css
ENV STRUCTURE_FILE=/data/output/directory-structure.txt
ENV CONTENT_FILE=/data/output/content.txt
ENV SUMMARY_FILE=/data/output/summary.txt

# Run the binary
# Use shell form to allow environment variable substitution
ENTRYPOINT ["/bin/sh", "-c"]
CMD ["/app/pack-files -dir=$INPUT_DIR -include=$INCLUDE_EXTS -exclude=$EXCLUDE_EXTS -structure=$STRUCTURE_FILE -content=$CONTENT_FILE -summary=$SUMMARY_FILE"]