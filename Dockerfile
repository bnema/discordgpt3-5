# Go building fie
FROM golang:1.18.0-alpine3.14

# Create a working directory
WORKDIR /app

COPY . ./

RUN mkdir -p /app/database

# Download the dependencies
RUN go mod download

# Install Git
RUN apk add --no-cache git

RUN apk add --no-cache gcc musl-dev


RUN go build

CMD ["./discordgpt3-5"]

# Expose port 80 and 443
EXPOSE 80 443