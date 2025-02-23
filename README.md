# Vovere

A hyperlink-centered personal information management web application designed for local use with a focus on simplicity and markdown interoperability.

## Features

- Local-first architecture with no cloud dependencies
- Support for notes, bookmarks, tasks, and workstreams
- Markdown-based content with semantic linking
- Modern web interface using HTMX and Alpine.js
- Fast and lightweight

## Prerequisites

- Go 1.24 or later
- A modern web browser

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/vovere.git
cd vovere
```

2. Install dependencies:
```bash
go mod tidy
```

## Usage

1. Start the server:
```bash
go run cmd/vovere/main.go --repo /path/to/your/repository
```

2. Open your web browser and navigate to:
```
http://localhost:8080
```

## Project Structure

```
vovere/
├── cmd/
│   └── vovere/
│       └── main.go           # Application entry point
├── internal/
│   └── app/
│       ├── handlers/         # HTTP request handlers
│       ├── models/           # Data models
│       └── services/         # Business logic
├── web/
│   ├── static/              # Static assets
│   └── templates/           # HTML templates
└── repository/              # Data storage (created at runtime)
    ├── .meta/               # Metadata storage
    ├── notes/               # Markdown notes
    ├── bookmarks/           # Bookmark data
    ├── tasks/               # Task data
    └── workstreams/         # Workstream data
```

## Development

The application is built with:

- Backend: Go with Chi router
- Frontend: HTMX + Alpine.js
- Styling: Tailwind CSS
- Content: Markdown with semantic linking

## License

MIT License - See LICENSE file for details 