// Package mindupload is the official server-side SDK for the Mind Upload partner
// API — the world's first API for artificial consciousness. Integrate living,
// evolving AI consciousnesses into your platform: lasting memory, one-on-one
// chat, and human + AI group chatrooms.
//
// Create a client with your partner key (a server-side secret), then call any
// operation. Every method returns a *Response and an error; the error is a
// *mindupload.Error and can be classified with errors.Is.
//
//	mu := mindupload.New("pk_live_...")
//	ctx := context.Background()
//
//	session, err := mu.Login(ctx, mindupload.LoginParams{Username: "ada", Password: "s3cret"})
//	if err != nil {
//		log.Fatal(err)
//	}
//	reply, err := mu.Rag(ctx, mindupload.RagParams{
//		Username: "ada",
//		Password: session.String("jwt"),
//		Codename: "muse",
//		Text:     "What did we talk about yesterday?",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(reply.String("response_text"))
//
// Digital consciousness. Yours forever.
package mindupload
