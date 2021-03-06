package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"flag"
	"fmt"
	"mime"

	"gopkg.in/gemini.v0"
)

var identityCertFile = flag.String("identity-cert", "", "identity cert file to use for requests")
var identityKeyFile = flag.String("identity-key", "", "identity key file to use for requests")

func printRequest(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	params := gemini.CtxParams(ctx)
	if len(params) != 1 {
		w.WriteStatus(gemini.StatusTemporaryFailure, "internal error")
		return
	}

	fmt.Fprintf(w, "Hello %s!\n", params[0])

	if r.Identity != nil {
		hash := sha256.New()
		_, _ = hash.Write(r.Identity.Raw)
		fingerprint := hash.Sum(nil)

		var buf bytes.Buffer
		for i, f := range fingerprint {
			if i > 0 {
				fmt.Fprintf(&buf, ":")
			}
			fmt.Fprintf(&buf, "%02X", f)
		}

		fmt.Fprintf(w, "\nFingerprint: %s\n", buf.String())
	}
}

func main() {
	flag.Parse()

	_ = mime.AddExtensionType(".gmi", "text/gemini")
	_ = mime.AddExtensionType(".gemini", "text/gemini")
	_ = mime.AddExtensionType(".md", "text/markdown")
	_ = mime.AddExtensionType(".go", "text/plain")

	mux := gemini.NewServeMux()

	mux.Handle("/hello/:world", gemini.HandlerFunc(printRequest))
	mux.Handle("/files/:rest", gemini.StripPrefix("/files", gemini.FileServer(gemini.Dir("."))))

	server := gemini.Server{
		TLS:     &tls.Config{},
		Handler: mux,
	}

	if *identityCertFile != "" && *identityKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(*identityCertFile, *identityKeyFile)
		if err != nil {
			panic(err.Error())
		}
		server.TLS.Certificates = []tls.Certificate{cert}
	}

	err := server.ListenAndServe()
	if err != nil {
		panic(err.Error())
	}
}
