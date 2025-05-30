# Chat CLI

A command-line client for the Chat Service that enables real-time messaging via the terminal.

---

## Features

- 🔐 User authentication with JWT tokens  
- 💬 Create and manage chats  
- 📨 Send and receive messages in real-time  
- 👥 Support for group chats  
- 🔄 Automatic token refresh  
- 📝 Detailed logging  
- 🖥️ Interactive command interface  

---

## Installation

```bash
# Clone the repository
git clone https://github.com/Mobo140/chat-cli.git
cd chat-cli

# Install dependencies
make install-deps

# Build the project
go build -o chat-cli main.go
````

---

## Usage

### Start the CLI

```bash
./chat-cli --config-path=path/to/config.env --log-level=info
```

---

## Commands

### Main Commands

#### 1. Login

```bash
login --username=username --password=password
```

Authenticate the user and create a session with JWT tokens upon success.

#### 2. Create Chat

```bash
create-chat --username user1 user2 user3...
```

Create a new chat with specified users.

- For a private chat: `create-chat --username john`
- For a group chat: `create-chat --username john alice bob`

#### 3. Connect to Chat

```bash
connect-chat --chat-id=ID --username=username
```

Connects to a chat and starts receiving messages in real-time.

- Use `Ctrl+C` to disconnect
- New messages will appear in the console

#### 4. Send Message

```bash
send-message --chat-id=ID Your message text here
```

Send a message to the specified chat.

- No quotes needed for messages with spaces
- Supports multiline messages

---

### Utility Commands

- `clear` — Clear the terminal screen
- `exit` / `quit` / `q` — Exit the application
- `help` — Show help for all commands
- `help command-name` — Show help for a specific command

---

## Usage Examples

1. Create and connect to a chat:

```bash
# Create a new chat
create-chat --username john alice
# Example output: ID: 29

# Connect to the chat
connect-chat --chat-id=29 --username=john

2. Send messages:

```bash
# Simple message
send-message --chat-id=29 Hello!

# Message with spaces
send-message --chat-id=29 Hello, how are you today?
```

---

## Token Refreshing

### Automatic Refresh

The app uses two JWT token types:

- `access_token` — short-lived (15 minutes)
- `refresh_token` — long-lived (24 hours)

#### Refresh Process

- Access token is refreshed automatically 1 minute before expiry
- Refresh token is refreshed automatically 1 hour before expiry

#### How it works

1. On successful login, user receives both tokens
2. Tokens are saved in the `.chat-cli-session` file
3. The system automatically:

   - Checks `access_token` expiry
   - If less than 1 hour left, uses `refresh_token` to get new tokens
   - Saves new tokens to the session file

#### Manual Refresh

Tokens can be refreshed manually by logging in again:

```bash
login --username=your_username --password=your_password
```

---

## Security

- Refresh tokens are stored encrypted
- Tokens can be invalidated by re-authentication
- All requests use secure (TLS) connections
