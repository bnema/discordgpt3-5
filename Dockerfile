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

# Define some ENV Vars
ENV DIRECTORY=/app \
  IS_DOCKER=true
  
CMD ["go", "run", "main.go"]

# Expose the port 443
EXPOSE 443