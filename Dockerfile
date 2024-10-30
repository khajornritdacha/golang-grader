# Dockerfile
FROM golang:latest

WORKDIR /app

# Copy the Go grader code
COPY main.go main.go

# Install g++
RUN apt-get update && apt-get install -y g++

# Build the grader binary
RUN go build -o main main.go

# Run the grader server
CMD ["./main"]
