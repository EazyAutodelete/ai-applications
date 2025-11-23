# AI Applications

A Discord bot application that generates AI-powered conversations across multiple channels using Google's Gemini AI.

## Overview

This project creates realistic Discord conversations using AI-generated messages from fictional characters. The bot manages multiple channels simultaneously, each with its own conversation history and independent message generation timers.

## Features

- **Multi-Channel Support**: Run conversations in up to 10 Discord channels concurrently
- **AI-Powered Conversations**: Uses Google Gemini 2.5 Flash Lite for natural language generation
- **Multiple Characters**: Five fictional characters (Zora, Kip, Luma, Dex, Nova) participate in conversations
- **Independent Channel State**: Each channel maintains its own conversation history and context
- **Dynamic Timing**: Messages are generated at varying intervals based on time of day
- **Staff Integration**: Staff members can inject messages into conversations using the `!ai` command

## Architecture

The application consists of several components:

### Conversation Generators (`conversation-generators/`)

The main application that generates and posts AI conversations:

- **main.go**: Core logic for message generation and multi-channel management
- **ai/**: AI client and generation logic for Google Gemini
- **gateway/**: Discord gateway bot for listening to staff messages

### AI Setup (`ai-setup/`)

Helper utilities for AI configuration and setup.

### Infrastructure (`infra/`)

Deployment and infrastructure configuration files.

## Configuration

The application requires the following environment variables:

### Discord Bot Tokens
- `DISCORD_TOKEN_0` through `DISCORD_TOKEN_4`: Bot tokens for the 5 character accounts
- `STAFF_ROLE`: Discord role ID for staff members who can interact with the bot

### Channel Configuration
- `CHANNEL_ID_0` through `CHANNEL_ID_9`: Discord channel IDs for the 10 conversation channels

### AI Configuration
- `AI_TOKEN`: Google Gemini API key

## How It Works

1. **Initialization**: The application starts 10 independent goroutines, one for each channel
2. **Message Generation**: Each channel's ticker generates messages at random intervals:
   - 2-5 minutes during peak hours (15:00-22:00 UTC)
   - 10-35 minutes during off-peak hours
3. **Character Selection**: A random character (different from the last speaker) is selected for each message
4. **AI Generation**: The AI generates a contextual response based on:
   - The last 3 messages in the conversation
   - The selected character's persona
   - Instructions to keep messages under 64 characters
5. **Posting**: The message is posted to Discord using the character's bot account
6. **History Management**: The message is added to the channel's history (keeping only the last 3 messages)

## Character Personas

The bot uses five fictional characters with distinct personalities:
- **Zora**
- **Kip**
- **Luma**
- **Dex**
- **Nova**

All characters share a humorous, nerdy, conversational tone and discuss technical, scientific, or geeky topics while avoiding sensitive or controversial subjects.

## Staff Interaction

Staff members with the configured role can inject messages into the conversation:

```
!ai Your message here
```

The bot will add this message to the conversation history, allowing the AI to respond to staff input.

## Building and Running

### Prerequisites
- Go 1.25.4 or later
- Discord bot accounts for each character
- Google Gemini API access

### Build
```bash
cd conversation-generators
go build -o conversation-generator
```

### Run
```bash
# Set environment variables
export AI_TOKEN="your-gemini-api-key"
export DISCORD_TOKEN_0="token-for-zora"
export DISCORD_TOKEN_1="token-for-kip"
export DISCORD_TOKEN_2="token-for-luma"
export DISCORD_TOKEN_3="token-for-dex"
export DISCORD_TOKEN_4="token-for-nova"
export CHANNEL_ID_0="channel-1-id"
export CHANNEL_ID_1="channel-2-id"
# ... set remaining channel IDs
export STAFF_ROLE="staff-role-id"

./conversation-generator
```

## Dependencies

Key dependencies include:
- `google.golang.org/genai`: Google Gemini AI client
- `github.com/disgoorg/disgo`: Discord API library
- `github.com/sirupsen/logrus`: Logging framework

See `conversation-generators/go.mod` for the complete dependency list.

## License

This project is part of the EazyAutodelete ecosystem.
