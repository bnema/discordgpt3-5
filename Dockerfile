FROM debian:bookworm-slim

WORKDIR /app

RUN mkdir /app/database

COPY discordgpt3-5 /app


CMD ["./discordgpt3-5"]

