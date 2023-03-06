FROM debian:bookworm-slim

WORKDIR /app

RUN mkdir /app/database

COPY discordgpt3-5 /app




RUN apt-get update && apt-get upgrade 


CMD ["./discordgpt3-5"]

