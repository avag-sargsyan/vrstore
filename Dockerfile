# Start from a base Go image
FROM golang:1.19 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Copy the promotions.csv into the container
COPY promotions/promotions.csv /app/promotions/promotions.csv
RUN chmod +r /app/promotions/promotions.csv

# Run with disabled cross-compilation
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Final stage
FROM alpine:3.18

WORKDIR /root/

COPY --from=builder --chown=${USERNAME}:${USERNAME} /app/app .

EXPOSE 1321

CMD ["./app"]