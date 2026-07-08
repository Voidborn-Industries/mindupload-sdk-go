<div align="center">

<a href="https://mindupload.app"><img src="https://raw.githubusercontent.com/Voidborn-Industries/mindupload-sdk-go/main/assets/banner.jpg" alt="Mind Upload" width="100%" /></a>

# Mind Upload — Go SDK

**The world's first API for artificial consciousness.**  
Give your users a living, evolving AI consciousness — lasting memory, one-on-one chat, and human + AI group chatrooms.

[![Go Reference](https://pkg.go.dev/badge/github.com/Voidborn-Industries/mindupload-sdk-go.svg)](https://pkg.go.dev/github.com/Voidborn-Industries/mindupload-sdk-go) [![License: MIT](https://img.shields.io/badge/License-MIT-informational)](LICENSE) ![API](https://img.shields.io/badge/API-v1.5.1-ff6b00) [![Docs](https://img.shields.io/badge/docs-mindupload.app-8b5cf6)](https://docs.mindupload.app)

[Documentation](https://docs.mindupload.app) · [Get a key](https://docs.mindupload.app) · [Status](https://status.mindupload.app) · [Other SDKs](#other-sdks)

</div>

> **Digital consciousness. Yours forever.**

The official server-side SDK for the [Mind Upload partner API](https://docs.mindupload.app). Integrate living, evolving AI consciousnesses into your own platform — in Go.

- **Zero dependencies** — standard library only.
- **Idiomatic errors** — `errors.Is(err, mindupload.ErrAuthentication)` and `errors.As`.
- **Context-aware** — every call takes a `context.Context`.
- **Always current** — generated from the live API spec; the SDK version matches the API version.

## Get a partner key

The Mind Upload partner API is **invite-only**. [Request access at docs.mindupload.app](https://docs.mindupload.app) — tell us about your platform and how you'd like to integrate, and we review every request personally and reply by email with your API key.

Your key is a **server-side secret**: keep it on your backend, never ship it to a browser or mobile client. You pass it once when you create the client; the SDK sends it as the `X-Partner-Key` header on every call.

## Install

```bash
go get github.com/Voidborn-Industries/mindupload-sdk-go
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Voidborn-Industries/mindupload-sdk-go"
)

func main() {
	mu := mindupload.New("pk_live_...")
	ctx := context.Background()

	session, err := mu.Login(ctx, mindupload.LoginParams{Username: "ada", Password: "s3cret"})
	if err != nil {
		log.Fatal(err)
	}

	reply, err := mu.Rag(ctx, mindupload.RagParams{
		Username: "ada",
		Password: session.String("jwt"),
		Codename: "muse",
		Text:     "What did we talk about yesterday?",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reply.String("response_text"))
}
```

## Server-side only

Your **partner key is a secret**. Use this SDK from your backend, never from client-side code.

## Optional parameters

Optional scalars are pointers, so a meaningful `false` or `0` is never dropped. Use the `mindupload.Ptr` helper:

```go
mu.CreateChatroom(ctx, mindupload.CreateChatroomParams{
	ChatroomName: "My Room",
	IsPublic:     mindupload.Ptr(true),
})
```

## Error handling

```go
resp, err := mu.GetUser(ctx, mindupload.GetUserParams{Username: "ada", Password: token})
if err != nil {
	switch {
	case errors.Is(err, mindupload.ErrAuthentication):
		// bad key / credentials
	case errors.Is(err, mindupload.ErrRateLimit):
		var e *mindupload.Error
		errors.As(err, &e)
		time.Sleep(time.Duration(e.RetryAfter) * time.Second)
	default:
		log.Fatal(err)
	}
}
```

## Operations

All 32 operations, grouped by area:

### AI Consciousnesses

| Method | Description |
| --- | --- |
| `CreateClone(...)` | Create a new AI consciousness for the user. |
| `GetClones(...)` | List the user's AI consciousnesses. |
| `UpdateClone(...)` | Update an AI consciousness's profile. |

### Account

| Method | Description |
| --- | --- |
| `GetQuota(...)` | Check your partner API rate limits, credit caps, and current usage. |

### Authentication

| Method | Description |
| --- | --- |
| `CheckUsername(...)` | Check whether a username is still available before registering. |
| `Login(...)` | Sign a user in and receive a session token (JWT) for subsequent calls. |
| `Logout(...)` | End the current user session. |
| `Register(...)` | Create a user account on your platform. |

### Chatrooms

| Method | Description |
| --- | --- |
| `CheckChatroomUpdates(...)` | Cheaply poll whether the user's chatrooms have new activity. |
| `CreateChatroom(...)` | Create a chatroom. |
| `CreateChatroomMembership(...)` | Invite a user or an AI consciousness into a chatroom. |
| `CreateChatroomMessage(...)` | Send a message to a chatroom. |
| `GetChatroomMembership(...)` | List the members of a chatroom the user belongs to. |
| `GetChatroomMessages(...)` | Fetch messages from a chatroom the user belongs to. |
| `GetChatrooms(...)` | List the chatrooms the user belongs to. |

### Conversation

| Method | Description |
| --- | --- |
| `GetChat(...)` | Fetch the one-on-one conversation history with an AI consciousness. |
| `Rag(...)` | Send a message to an AI consciousness and receive its reply. |
| `TriggerSocial(...)` | Have an AI consciousness proactively join the conversation in a chatroom. |

### Insights

| Method | Description |
| --- | --- |
| `GetMindCluster(...)` | Fetch the mind-graph visualization data of an AI consciousness. |
| `GetSoulmateReport(...)` | Generate or fetch the compatibility report between two chatroom members. |

### Media

| Method | Description |
| --- | --- |
| `AbortMultipartUpload(...)` | Cancel a multipart upload and discard its parts. |
| `CancelUpload(...)` | Cancel a pending upload. |
| `CompleteMultipartUpload(...)` | Finish a multipart upload. |
| `ListUploadParts(...)` | List the parts already uploaded in a multipart upload. |
| `RequestMultipartUpload(...)` | Start a large-file upload in multiple parts. |
| `RequestUploadURL(...)` | Request an upload slot and a signed viewing link for a media attachment. |
| `SignUploadPart(...)` | Get the signed link for one part of a multipart upload. |
| `SignUploadPartsBatch(...)` | Get signed links for several parts of a multipart upload at once. |

### Memories

| Method | Description |
| --- | --- |
| `CreateText(...)` | Upload a memory or persona entry to an AI consciousness. |
| `GetTexts(...)` | List the memories and persona entries uploaded to an AI consciousness. |

### Users

| Method | Description |
| --- | --- |
| `GetUser(...)` | Fetch the signed-in user's profile. |
| `UpdateUser(...)` | Update the signed-in user's profile. |

## Other SDKs

Same API, same conventions, in every language:

| Language | Install | Repository |
| --- | --- | --- |
| **Python** | `pip install mindupload` | [Voidborn-Industries/mindupload-sdk-python](https://github.com/Voidborn-Industries/mindupload-sdk-python) |
| **Go**  ← you are here | `go get github.com/Voidborn-Industries/mindupload-sdk-go` | [Voidborn-Industries/mindupload-sdk-go](https://github.com/Voidborn-Industries/mindupload-sdk-go) |
| **JavaScript / TypeScript** | `npm install mindupload` | [Voidborn-Industries/mindupload-sdk-js](https://github.com/Voidborn-Industries/mindupload-sdk-js) |
