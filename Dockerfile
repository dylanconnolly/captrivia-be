# Builder stage
FROM golang:1.22.2-alpine3.19 as builder
WORKDIR /app

COPY . .

# RUN go install github.com/air-verse/air@latest
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build


EXPOSE 8080

# Command to run the executable
# CMD ["air", "--", "-listen", ":8080"]
CMD ["./captrivia-be"]
