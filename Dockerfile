# Use the official Go image as the base image
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download and cache Go modules
RUN go mod download

# Copy the agent source code
COPY main.go ./

# Build the agent binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent .

# Create a minimal Docker image to run the agent
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /app/agent .

# Run the agent binary when the container starts
CMD ["./agent"]
