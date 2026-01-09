# Vertex Go SDK

Unofficial Go SDK for [Vertex](https://github.com/vertex-app/vertex).

## Installation

```bash
go get github.com/your-username/vertex-go-sdk
```

## Usage

### Authentication

```go
package main

import (
	"log"
	"vertex-sdk"
)

func main() {
	client, err := vertex.NewClient("http://127.0.0.1:3000")
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Login("admin", "password"); err != nil {
		log.Fatal(err)
	}

	// Now you can call API methods
}
```

### List RSS Tasks

```go
rssList, err := client.ListRss()
if err != nil {
    log.Fatal(err)
}
for _, rss := range rssList {
    fmt.Printf("RSS: %s (Enable: %v)\n", rss.Alias, rss.Enable)
}
```

### Manage Rules

```go
// List RSS Rules
rules, err := client.ListRssRules()

// List Delete Rules
delRules, err := client.ListDeleteRules()
```

### View History

```go
history, err := client.ListRssHistory(1, 10, "")
```

## Features

- Server Management
- Downloader Management
- RSS Task Management
- RSS/Select Rule Management
- Delete Rule Management
- Torrent Management & History
- Monitoring (NetSpeed, CPU, Memory, etc.)

## License

MIT
