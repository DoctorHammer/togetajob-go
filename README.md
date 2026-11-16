# togetajob-go

The official Go client for [togetajob.com](https://togetajob.com), a free,
independent guide to New York City government jobs and civil service exams.

No account, no key, no signup. Public reads are rate-limited to 100 requests
a minute per caller; a free API key raises that to 600 a minute.

## Install

```sh
go get github.com/DoctorHammer/togetajob-go
```

Requires Go 1.23 or newer. The module has no dependencies.

## Use

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DoctorHammer/togetajob-go"
)

func main() {
	client := togetajob.NewClient()
	ctx := context.Background()

	health, err := client.Health(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("last refresh:", health.LastRefresh)

	exams, err := client.Exams(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range exams.Exams {
		fmt.Println(e.Number, e.Title, "filing closes", e.FilingCloses)
	}

	jobs, err := client.Jobs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(jobs.Rows), "open postings")

	posting, err := client.Posting(ctx, "808867")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(posting.Title, "at", posting.Agency)
}
```

## API keys

A key is optional and free, and it only raises the rate limit. Create one
with no sign-in:

```go
key, err := client.CreateKey(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println(key.Key) // shown once; only its hash is stored server-side
```

Then pass it to the client:

```go
client := togetajob.NewClient(togetajob.WithAPIKey("tj_live_..."))
```

Keys expire a year after creation and can be revoked:

```go
err := client.RevokeKey(ctx, "tj_live_...")
```

## What the client wraps

| Method | Endpoint | Returns |
| --- | --- | --- |
| `Health` | `GET /api/health` | API status and last data refresh |
| `Exams` | `GET /data/exams.json` | Civil service exam calendar |
| `Guides` | `GET /data/guides.json` | Step-by-step job guides |
| `Jobs` | `GET /jobs/index.json` | Every open posting, easiest first |
| `Posting` | `GET /api/posting?id=` | One open posting by city job ID |
| `CreateKey` | `POST /api/keys` | A free API key, shown once |
| `RevokeKey` | `DELETE /api/keys` | Revokes a key |

The full API contract is at https://togetajob.com/data/openapi.json and the
developer docs at https://togetajob.com/developers. The MCP server lives at
https://togetajob.com/mcp.

## License

MIT. See [LICENSE](LICENSE).
