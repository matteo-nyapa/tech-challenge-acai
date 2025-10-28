# 🌴 Acai Technical Challenge – AI Assistant (Go)

This project is my solution to the **Acai Travel Technical Challenge** for the *Software Engineer* position.

The app is a personal assistant service similar to ChatGPT, allowing you to have conversations with an AI assistant and obtain useful information, such as the weather or holidays, through a **Twirp** API.
During the challenge, **all 5 proposed tasks** were completed, with bonuses included.

---

## 🚀 Core Technologies

- **Go** (1.22+)
- **MongoDB** (Docker)
- **Twirp** (gRPC/JSON framework)
- **OpenAI API**
- **OpenTelemetry** (metrics and traces)
- **WeatherAPI** (real-world weather data)
- **Gorilla/Mux** (routing)
- **Testify** and `httptest` for unit tests

---

## ⚙️ Setup and Run

### 1. Clone the repository

```bash
git clone https://github.com/matteo-nyapa/acai-challenge.git
cd acai-challenge
```

### 2. Required Environment Variables

Create a `.env` file or export variables directly:

```bash
export OPENAI_API_KEY=your_openai_api_key
export WEATHER_API_KEY=your_weatherapi_api_key
```

> 💡 You can get a free API key at [WeatherAPI.com](https://www.weatherapi.com/).

### 3. Start MongoDB and run the server

Make sure you have Docker running and run:

```bash
make up run
```

The server will start at:

👉 [http://localhost:8080/](http://localhost:8080/).

---

## 💬 Main API

### **POST /twirp/acai.chat.ChatService/StartConversation**

Start a new conversation with the AI ​​assistant.
This endpoint creates a conversation in MongoDB, automatically generates a title, and returns the assistant's first response.

#### 🧠 Request Example

```bash
curl -s -X POST 'http://localhost:8080/twirp/acai.chat.ChatService/StartConversation' \
-H 'Content-Type: application/json' \
-d '{"message":"What is the weather in Barcelona today?"}' | jq .
```

#### 📦 Response Example

```json
{
"conversation_id": "68a6e63c288abccdf52b6355",
"title": "Weather in Barcelona",
"reply": "Currently in Barcelona: 20.4°C, Sunny, wind 10 km/h. Forecast available for the next 3 days."
}
```

---

## 🧠 Wizard Features

The wizard uses a modular system of **tools** that extend its capabilities beyond text, allowing it to access external data or perform specific functions.

The currently active tools are listed below:

| Tool | Description |
|------|--------------|
| 🗓️ `get_today_date` | Returns the current date and time in RFC3339 format |
| ☀️ `get_weather` | Query the current weather or forecast using the WeatherAPI |
| 🎉 `get_holidays` | Displays official holidays for Barcelona (remote ICS file) |
| ⏰ `time_in` | Returns the current time in a specific time zone *(bonus tool)* |

> These tools are dynamically registered using a **registry**, allowing new tools to be added without modifying the assistant's main code.

---

## 🧪 Tests

The tests cover both the server and the assistant tools.
They include **unit and integration** tests that ensure the correct operation of each component.

### 🔍 Main Coverage

- **Tools:**
- `weather`, `holidays`, `today`, `time_in`, `registry`
- **API:**
- `StartConversation` → creates a conversation, assigns a title, and generates a response.
- **Assistant:**
- `Title` → generates concise and relevant titles.
- **Weather client:**
- Use of a mock HTTP server (no external dependencies).

### ▶️ Run all tests

Make sure you have MongoDB running (use `make up`) and then run:

```bash
go test ./... -v
```

#### 🧾 Sample output

```bash
=== RUN TestWeatherTool_Call_UsesMockWeatherAPI
--- PASS: TestWeatherTool_Call_UsesMockWeatherAPI (0.01s)
=== RUN TestServer_StartConversation_CreatesConversation
--- PASS: TestServer_StartConversation_CreatesConversation (0.03s)
PASS
ok github.com/usuario/acai-challenge/internal/... 1.2s
```

---

## 📊 Observability (Task 5)

Added **metrics and tracing** using **OpenTelemetry** to gain visibility into server performance.

### 📈 Collected Metrics

| Metric | Type | Description |
|----------|------|--------------|
| `http.server.requests` | Counter | Total number of requests received |
| `http.server.duration.seconds` | Histogram | Average request duration |
| `http.server.errors` | Counter | Total number of errors logged |

### 🧩 Tracing

Each Twirp request generates a **span** called `twirp.request` with attributes that describe the request's execution flow:

- `rpc.system`: `"twirp"`
- `rpc.service`: `"ChatService"`
- `rpc.method`: `"StartConversation"`
- `http.route`: `"ChatService/StartConversation"`
- `http.status_code`: `"200"`

#### 🧾 Console output example (stdout exporter)

```json
{
"Name": "twirp.request",
"Attributes": [
{"Key": "rpc.service", "Value": "ChatService"},
{"Key": "rpc.method", "Value": "StartConversation"},
{"Key": "http.status_code", "Value": "200"}
],
"StartTime": "2025-10-28T15:00:11Z",
"EndTime": "2025-10-28T15:00:15Z"
}
```

> 💡 **Note:**
> Metrics and traces are exported in **JSON** format directly to standard output (`stdout`).
> This allows you to **monitor response times, errors, and execution flow** without needing to configure an additional backend
> (such as **Jaeger**, **Grafana**, or **Prometheus**).

---

## 📁 Relevant structure of the project

```bash
cmd/
├── cli/ 
│ └── main.go
└── server/
└── main.go
internal/
├── chat/
│ ├── assistant/ 
│ │ ├── assistant.go 
│ │ ├── assistant_test.go
│ │ ├── calendar/ 
│ │ └── tools/ 
│ ├── model/ 
│ │ ├── conversation.go
│ │ ├── message.go
│ │ ├── repository.go
│ │ ├── role.go
│ │ └── testing/ 
│ │ ├── fixture.go
│ │ ├── mongo.go
│ │ └── server_test.go
│ └── server.go 
├── httpx/ 
│ ├── logger.go
│ └── recovery.go
├── mongox/ 
│ └── connect.go
├── observability/ 
│ ├── otel.go
│ └── setup.go
├── pb/ 
│ ├── chat.pb.go
│ └── chat.twirp.go
└── weather/ 
├── weather.go
└── weather_test.go
rpc/
└── chat.proto # definición del servicio Twirp/Protobuf
docker-compose.yaml 
go.mod 
go.sum
Makefile 
README.md 
```
---

## ✅ Completed Tasks

| Task | Description | Status |
|------|--------------|:------:|
| 1 | Fix conversation title | ✅ |
| 2 | Fix the weather (real API + forecast) | ✅ |
| 3 | Refactor tools (registry + bonus tool) | ✅ |
| 4 | Test StartConversation API + Title | ✅ |
| 5 | Add OpenTelemetry metrics + tracing | ✅ |

---

## 🏁 Submission

Public repository:
👉 **https://github.com/matteo-nyapa/acai-challenge**

Includes:
- Complete code (`go.mod`, `Makefile`, `docker-compose.yml`, etc.)
- Working tests (`go test ./...`)
- Updated README (this file)
- Metrics and traces visible in console (stdout)
