# Builder stage
FROM golang:1.22.2-alpine3.19 as builder
WORKDIR /app

COPY . .

RUN go install github.com/air-verse/air@latest

EXPOSE 8080

# Command to run the executable
CMD ["air", "--", "-listen", ":8080"]
