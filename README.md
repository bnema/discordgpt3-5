# Discord Bot with ChatGPT and Golang

This project is a fork of [navicstein/telechatgpt](https://github.com/navicstein/telechatgpt) that has been updated for Discord.

## Explanation

While the original project was tailored to Telegram and only allowed for one-on-one conversations, this version has been adapted for Discord and can handle conversations with multiple users at the same time (can be buggy at times).

Also, the retain history has to be limited (set to 10 + system prompt) to not go over quota too fast.

## Usage

For easier deployment, instead of having the system prompt in a txt file it is now stored alone in a table "system_prompt" in the database (Only 1 entry is allowed).

Once the bot is running, you can use the command `!systemprompt + your prompt` to set the system prompt. (You can change the system prompt at any time to try different things.)

You can reset the database with the command `!resetdb`.

## Required environment variables

```bash
OPENAI_TOKEN= # OpenAI API key
DISCORD_APP_ID= # Discord App ID
DISCORD_BOT_TOKEN= # Discord Bot Token
DISCORD_GUILD_ID= # Discord Guild ID
DISCORD_CHANNEL_ID= # Discord Channel ID (where the bot will be used)
RETAIN_HISTORY= # boolean, whether to retain history or not
```

## Todo

-   [ ] isTyping indicator when waiting for openai's response
-   [ ] Add a wait delay to prevent spamming the bot (also concat if the user sends multiple messages in a row)
-   [ ] Listen in every channels but add some randomness to decide if the bot should respond or not
-   [ ] Add role check for the commands
-   [ ] Try to make the bot arbitrarly use discord commands like `/timeout user 1m` for more hilarious situations