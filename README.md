# gemini

This is an in-progress gemini client and server package for go. It aims to mirror the standard library's `net/http` package.

The import url is `gopkg.in/gemini.v0`.

## Gemini Spec

This package aims to implement version `0.14.3` of the [Gemini spec](https://gemini.circumlunar.space/docs/specification.html) [(gemini://)](gemini://gemini.circumlunar.space/docs/specification.gmi)

## Status

- [ ] Client implementation
    - [x] Basic request
    - [ ] Client auth
- [ ] Server implementation
    - [ ] Unsafe implementation (no TLS, unfortunately no client auth)
    - [ ] TLS implementation
    - [ ] Basic routing
    - [ ] FileSystem implementation, possibly integrating with Go 1.16's FS.
- [ ] Gemtext implementation
    - [ ] Parser
    - [ ] Writer
