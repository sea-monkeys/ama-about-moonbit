FROM golang:1.24.0-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git

# Copy all source code files
COPY . .

RUN <<EOF
go mod download
go mod tidy 
go build -o create-embeddings
EOF

FROM scratch
WORKDIR /app
COPY --from=builder /app/create-embeddings .

CMD ["./create-embeddings"]