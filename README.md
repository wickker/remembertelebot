# RememberOrDismember Bot 🐟

A Telegram bot that helps you remember things 🗡️

Try it: [@remember_or_dismember_bot](https://t.me/remember_or_dismember_bot)

## Features

- **One-time Reminders**: Set reminders for specific dates and times
- **Recurring Reminders**: Set up periodic reminders with cron-like scheduling
- **AI-Powered Conversations**: Powered by DeepSeek AI to infer cron expressions from natural language
- **Job Management**: Create, list, and cancel reminder jobs
- **Webhook Support**: Receives updates via webhooks for better performance
- **Graceful Shutdown**: Proper cleanup of resources and background jobs

## Commands

- `/start` - Show help menu and bot introduction
- `/newjob` - Create a new reminder job (guided setup)
- `/listjobs` - List all your active reminder jobs
- `/canceljob-<jobID>` - Cancel a specific job (e.g., `/canceljob-123`)

## Prerequisites

- Go 1.24+ 
- PostgreSQL database
- Telegram Bot Token (from [@BotFather](https://t.me/botfather))
- DeepSeek API Key
- sqlc for database code generation

### Install Dependencies

```bash
# Install sqlc for database code generation
brew install sqlc

# Install Go dependencies
go mod vendor && go mod tidy
```

## Environment Setup

Create a `.env` file in the project root with the following variables:

```env
ENV=dev
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
DATABASE_URL=your_db_url_here
BASE_URL=https://your-domain.com
DEEP_SEEK_API_KEY=your_deepseek_api_key_here
```

## Database Setup

The bot uses PostgreSQL with the following tables:
- `chats`: Stores chat information and context
- `jobs`: Stores reminder jobs with scheduling information

Database migrations are handled via SQL schema files in `db/schemas/`.

## Local Development

1. **Generate database code**:
   ```bash
   sqlc generate
   ```

2. **Set up your environment**:
   - Ensure your PostgreSQL database is running and accessible

3. **Run the bot**:
   ```bash
   go run main.go
   ```

4. **Set up webhook** (for local development):
   - Use ngrok or similar tool to expose your local server
   - Update `BASE_URL` in your `.env` file

## Build and Deploy

### Google Cloud Run Deployment

1. Follow the basic setup here: https://cloud.google.com/run/docs/quickstarts/build-and-deploy/deploy-go-service

2. Amend `cloudbuild.yaml` with `gcr.io/<PROJECT_ID>/<YOUR_SERVICE_NAME>`

3. Run `gcloud builds submit` at the project root

4. Navigate to Cloud Run on the GC Console and deploy the submitted build

### Docker

The project includes a `Dockerfile` for containerized deployment:

```bash
docker build -t remembertelebot .
docker run -p 9000:9000 --env-file .env remembertelebot
```

## Architecture

- **Bot Framework**: Telegram Bot API with webhook support
- **Database**: PostgreSQL with sqlc for type-safe queries
- **Job Scheduling**: River queue for background job processing
- **AI Integration**: DeepSeek AI for conversational features
- **Caching**: Ristretto for in-memory caching
- **Logging**: Structured logging with zerolog

## Project Structure

```
├── bot/                 # Telegram bot client
├── config/             # Configuration management
├── db/                 # Database schemas and generated code
│   ├── queries/        # SQL queries
│   ├── schemas/        # Database schema files
│   └── sqlc/          # Generated Go code
├── services/           # Business logic
│   ├── commands/       # Command handlers
│   ├── messages/       # Message handlers
│   └── callbackqueries/ # Callback query handlers
├── riverjobs/          # Background job processing
├── deepseekai/         # AI integration
├── ristrettocache/     # Caching layer
└── main.go            # Application entry point
```

## License

This project is open source and available under the [MIT License](LICENSE).