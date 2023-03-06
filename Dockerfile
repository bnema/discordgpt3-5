# Go building fie
FROM golang:1.20.1-bullseye

# Create a working directory
WORKDIR /app

# Copy the application source code to the container
COPY . .

# Create the database directory
RUN mkdir -p /app/database

# Install the dependencies
RUN go get -d -v ./...
RUN go install -v ./...

# Expose port 80 and 443
EXPOSE 80 443

# Start the application
CMD ["go", "run", "main.go"]