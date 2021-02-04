# gemini

[![GoDoc](https://img.shields.io/badge/doc-GoDoc-007d9c.svg)](https://pkg.go.dev/gopkg.in/gemini.v0)

This is an in-progress gemini client and server package for go. The design is based on a combination of `net/http` and [chi](https://github.com/go-chi/chi).

The import url is `gopkg.in/gemini.v0`.

## Gemini Spec

This package aims to implement version `0.14.3` of the [Gemini spec](https://gemini.circumlunar.space/docs/specification.html) ([gemini://gemini.circumlunar.space/docs/specification.gmi](gemini://gemini.circumlunar.space/docs/specification.gmi))

## Usage

### Client

```go
package main

import "gopkg.in/gemini.v0"

func main() {
    client := &gemini.Client{}

    resp, err := client.Get("gemini.circumlunar.space")
    if err != nil {
        panic(err.Error())
    }

    fmt.Println(resp.Status, resp.Meta)

    if !resp.IsSuccess() {
        return
    }

    fmt.Println()

    _, err = io.Copy(os.Stdout, resp.Body)
    if err != nil {
        panic(err.Error())
    }
}
```

### Server

```go
package main

import (
    "context"
    "crypto/tls"
    "fmt"
    "mime"

    "gopkg.in/gemini.v0"
)

func main() {
    _ = mime.AddExtensionType(".gmi", "text/gemini")
    _ = mime.AddExtensionType(".gemini", "text/gemini")

    mux := gemini.NewServeMux()

    // A simple dynamic handler
    mux.Handle("/hello/:world", gemini.HandlerFunc(func (ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
        params := gemini.CtxParams(ctx)
        if len(params) != 1 {
            gemini.WriteStatus(gemini.StatusCGIError, "internal error")
            return
        }

        fmt.Fprintf(w, "Hello %s\n", params[0])
    }))

    // Serve all files from `/tmp` out of the path `/files` via gemini.
    mux.Handle(
        "/files/:rest",
        gemini.StripPrefix("/files/", gemini.FileServer(gemini.Dir("/tmp"))),
    )

    cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
    if err != nil {
        panic(err.Error())
    }

    server := gemini.Server{
        TLS: &tls.Config{
            ClientAuth:         tls.RequestClientCert,
            Certificates:       []tls.Certificate{cert},
        },
        Handler: mux,
    }

    err := server.ListenAndServe()
    if err != nil {
        panic(err.Error())
    }
}
```

## Feature Status

- [x] Client implementation
    - [x] Basic request
    - [x] Client auth
    - [x] Proxy request
    - [ ] TOFU
- [x] Server implementation
    - [x] TLS implementation
    - [x] Basic routing
    - [x] FileSystem implementation, based on net/http.
    - [ ] Add logging interface
    - [ ] Basic middleware - logging, recoverer
    - [ ] Integrate FileSystem with Go 1.16's FS.
    - [ ] Conveniences for dealing with client certs
    - [ ] Routing based on SNI
    - [ ] Routing based on request URL protocol and hostname (for proxy support)
- [ ] Gemtext implementation
    - [ ] Parser
    - [ ] Writer
- [ ] API cleanup
    - [x] Simplify TLS cert handling
    - [x] Switch to a ResponseWriter pattern
- [ ] Various cleanup
    - [ ] Add tests
