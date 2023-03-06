# Go building fie
FROM golang:1.18.0-alpine3.14

# Create a working directory
WORKDIR /app

# Copy all the files from the current directory to the working directory
COPY . ./


# Download the dependencies
RUN go mod download

# Install Git
RUN apk add --no-cache git

# Build the Go app with CGO disabled and statically linked
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o discordgpt-linux-amd64 .


# Define some ENV Vars
ENV DIRECTORY=/app \
  IS_DOCKER=true

CMD ["./discordgpt-linux-amd64"]

# Expose the port 443
EXPOSE 443