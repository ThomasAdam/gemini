# gemini

This is an in-progress gemini client and server package for go. It aims to mirror the standard library's `net/http` package.

The import url is `gopkg.in/gemini.v0`.

## Gemini Spec

This package aims to implement version `0.14.3` of the [Gemini spec](https://gemini.circumlunar.space/docs/specification.html) [(gemini://)](gemini://gemini.circumlunar.space/docs/specification.gmi)

## Status

- [x] Client implementation
    - [x] Basic request
    - [x] Client auth
- [ ] Server implementation
    - [x] TLS implementation
    - [x] Basic routing
    - [ ] FileSystem implementation, possibly integrating with Go 1.16's FS.
- [ ] Gemtext implementation
    - [ ] Parser
    - [ ] Writer
