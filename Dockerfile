# Go building fie
FROM golang:1.20.1-bullseye

# Create a working directory
WORKDIR /app

COPY . ./

RUN mkdir -p /app/database

# Download the dependencies
RUN go mod download

RUN go mod tidy


# Expose port 80 and 443
EXPOSE 80 443

CMD ["go", "run", "main.go"]

