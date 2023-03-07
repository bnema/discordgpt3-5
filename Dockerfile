# Go building fie
FROM golang:1.18.0-alpine3.14

# Create a working directory
WORKDIR /app

# Copy all the files from the current directory to the working directory
COPY . ./

RUN mkdir -p /app/database

# Download the dependencies
RUN go mod download

# Install Git
RUN apk add --no-cache git

RUN apk add --no-cache gcc musl-dev

# Build the Go app with CGO enabled 
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o discordgpt-linux-amd64 .


# Define some ENV Vars
ENV DIRECTORY=/app \
  IS_DOCKER=true

# Expose the port 80 and 443
EXPOSE 80 443

CMD ["./discordgpt-linux-amd64"]

