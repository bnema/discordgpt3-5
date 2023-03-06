# Go building fie
FROM golang:1.18.0-alpine3.14

# Create a working directory
WORKDIR /app

COPY . ./

RUN mkdir -p /app/database

# Download the dependencies
RUN go mod download

RUN go build

CMD ["./discordgpt3-5"]

