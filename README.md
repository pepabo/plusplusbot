# plusplusbot

plusplusbot is a Slack bot that runs as a [Slack App](https://api.slack.com/docs/apps) to award points to users in your Slack workspace. You can give points to someone by typing `@username++` when you want to express gratitude or celebrate their achievements. For example, when someone makes a great suggestion, successfully leads a project, or helps you out in a difficult situation, you can show your appreciation by giving them points.

This project is inspired by [pluspl.us](https://github.com/plusplusslack/pluspl.us) (now archived), a similar Slack bot that allows users to reward team members with imaginary points.

## Features

- `@username++` - Add 1 point to the specified user
- `@username--` - Subtract 1 point from the specified user
- `@username==` - Check the current points of the specified user

## Slack App Configuration

The following settings are required to run this bot:

### Basic Settings
- Socket Mode: Enable
  - Enable Event Subscriptions in the Socket Mode settings page

### Required Tokens and Permissions
- App-Level Token
  - `connections:write` (for Socket Mode)
- Bot Token Scopes
  - `app_mentions:read` (to read mentions)
  - `channels:history` (to read channel message history)
  - `chat:write` (to send messages)
  - `user:read` (to read user information)
- Event Subscriptions
  - Bot Events
    - `message.channels` (to handle channel messages)

See our example [slack-app-manifest.json](slack-app-manifest.json) for more details.

## Setup

### Required Environment Variables

- `SLACK_BOT_TOKEN` - Slack bot token (starts with `xoxb-`)
- `SLACK_APP_TOKEN` - Slack app token (starts with `xapp-`)
- `DATABASE_URL` - Database file path
- `DEBUG` - Set any value to enable debug mode

### Database

This bot uses SQLite database to persist points. The database file path is specified by the `DATABASE_URL` environment variable.

#### Example Configuration
```
DATABASE_URL=file://plusplus.db
```
With the above configuration, a `plusplus.db` file will be created in the current directory.

### Installation

```bash
go mod download
go build
```

### Running

```bash
./plusplusbot
```

## License

MIT
