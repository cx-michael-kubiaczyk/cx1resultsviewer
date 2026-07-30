# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...
go run main.go
go test ./internal/backend/
go vet ./...
go mod tidy
```

`main.go` reads Checkmarx One credentials from environment variables via `Cx1ClientGo.NewClient()`. Check the `Cx1ClientGo` library docs for the exact variable names (`CX_BASE_URI`, `CX_TENANT`, `CX_CLIENT_ID`, `CX_CLIENT_SECRET`, etc.).

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
