# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...
go run . -apikey %CX1_DEU_KEY%
go test ./internal/backend/
go vet ./...
go mod tidy
```

The `-apikey` flag provides the Checkmarx One API key. Additional parameters can be provided directly on the command-line, visible by running with the -h parameter:

```
  -apikey string
        CheckmarxOne API Key (if not using client id/secret)
  -client string
        CheckmarxOne Client ID (if not using API Key)
  -cx1 string
        If using client id and secret: CheckmarxOne platform URL
  -iam string
        If using client id and secret: CheckmarxOne IAM URL
  -log string
        Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL (default "INFO")
  -secret string
        CheckmarxOne Client Secret (if not using API Key)
  -tenant string
        If using client id and secret: CheckmarxOne tenant
  -token string
        Alternative: A valid access_token. If this value is provided, others will be ignored - the client will lose access when the token expires
  -useragent string
        Optional: A custom user-agent string for all requests to Cx1 (default "Cx1ClientGo")
```

## Architecture

This is an early-stage CLI prototype that fetches Checkmarx One SAST scan results and return the relevant source files.

**Module:** `cx1resultsviewer`  
**Key dependency:** `github.com/cxpsemea/Cx1ClientGo` — the Checkmarx One API client

### Data flow

```
main.go
  → Cx1ClientGo.NewClient()        # authenticates via env vars
  → backend.NewServer()
  → server.Run() → server.test()   # hard-coded proof-of-concept URL
      → LoadResult(url)
          → extractIDFromURL()     # parses /sast-results/{projectID}/{scanID}?resultId=...
          → createCodeExtract()    # fetches results + source from CX1 API
              → GetAllScanSASTResultsFiltered()
              → GetScannedFileSourceByID() per result node
              → CodeSet.AddFile() / AugmentFile()
      → CodeSet.GetSources()       # prints source to stdout
```

### Core types (`internal/backend/`)

- **`FileSource`** — holds source code
- **`CodeSet`** — `map[filePath]string`; the full source set for a scan result.
- **`WebServer`** — top-level struct holding the `Cx1Client`, logger, and `CodeSet`. Named for a planned HTTP server that is not yet built; currently runs as a one-shot CLI.
- **`util.go`** — URL parsing (`extractIDFromURL`) and the main API orchestration (`createCodeExtract`).

The `WebServer.test()` method in `backend.go` contains the hard-coded result URL used during development — this is the expected entry point for experimenting with new functionality until a real HTTP server or CLI interface is wired up.
