# testcontainers-go-resend

A [Testcontainers for Go](https://golang.testcontainers.org/) module that provides a mock [Resend](https://resend.com) API for integration testing.

It uses [Microcks](https://microcks.io/) under the hood to serve mock responses based on the official [Resend OpenAPI spec](https://github.com/resend/resend-openapi).

## Install

```bash
go get github.com/mdelapenya/testcontainers-go-resend
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/testcontainers/testcontainers-go"

	"github.com/mdelapenya/testcontainers-go-resend"
)

func main() {
	ctx := context.Background()

	ctr, err := resend.Run(ctx, resend.DefaultImage)
	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		log.Fatal(err)
	}

	baseURL, err := ctr.BaseURL(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Resend mock API available at:", baseURL)
}
```

## Custom OpenAPI spec URL

By default the module fetches the latest Resend OpenAPI spec from GitHub. If the download fails, it falls back to an embedded copy.

You can point to a different spec URL:

```go
ctr, err := resend.Run(ctx, resend.DefaultImage,
	resend.WithSpecURL("https://example.com/my-resend-spec.yaml"),
)
```

## Supported endpoints

The mock covers all Resend API resources: emails, domains, API keys, templates, audiences, contacts, broadcasts, webhooks, segments, topics, contact properties, contact segments, and contact topics.

## License

[MIT](LICENSE)
