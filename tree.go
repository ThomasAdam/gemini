package gemini

import (
	"context"
	"fmt"
	"path"
	"strings"
)

type nodeType uint8

const (
	ntStatic nodeType = iota
	ntParam
	ntCatchAll
)

type node struct {
	handler         Handler
	catchAllHandler Handler
	children        map[string]*node
	param           *node
}

func newNode() *node {
	return &node{children: make(map[string]*node)}
}

func (n *node) ServeGemini(ctx context.Context, r *Request) *Response {
	fmt.Println(r.URL.Path)

	params, handler := n.match(r.URL.Path)
	if handler == nil {
		return nil
	}

	ctx = CtxWithParams(ctx, params)
	return handler.ServeGemini(ctx, r)
}

func (n *node) Handle(pattern string, h Handler) {
	target := n.ensureNode(pattern)
	if target.handler != nil {
		panic("overlapping handlers")
	}
	target.handler = h
}

func (n *node) NotFound(h Handler) {
	target := n.ensureNode(":rest")
	if target.catchAllHandler != nil {
		panic("overlapping catchAllHandlers")
	}
	target.catchAllHandler = h
}

func (n *node) Route(pattern string, fn func(r Router)) Router {
	target := n.ensureNode(pattern)
	fn(target)
	return target
}

func (n *node) ensureNode(targetPath string) *node {
	return n.ensureNodeImpl(strings.Trim(path.Clean(targetPath), "/"))
}

func (n *node) ensureNodeImpl(path string) *node {
	if path == "" {
		return n
	}

	next, rest := pathSegment(path)

	// As a special case, we want to have a catch-all option if the last param
	// is named :rest
	if next == ":rest" && rest == "" {
		return n
	}

	if strings.HasPrefix(next, ":") {
		if n.param == nil {
			n.param = newNode()
		}

		return n.param.ensureNodeImpl(rest)
	}

	target := n.children[next]
	if target == nil {
		n.children[next] = newNode()
		target = n.children[next]
	}

	return target.ensureNodeImpl(rest)
}

func (n *node) match(targetPath string) ([]string, Handler) {
	return n.matchImpl(strings.Trim(path.Clean(targetPath), "/"), nil)
}

func (n *node) matchImpl(path string, params []string) ([]string, Handler) {
	if n == nil {
		return nil, nil
	}

	if path == "" {
		return params, n.handler
	}

	next, rest := pathSegment(path)

	// First attempt static routes.
	retParams, retHandler := n.children[next].matchImpl(rest, params)
	if retHandler != nil {
		return retParams, retHandler
	}

	// If there isn't a matching static route, attempt a param route.
	retParams, retHandler = n.param.matchImpl(rest, append(params, next))
	if retHandler != nil {
		return retParams, retHandler
	}

	// Finally fall back to the catch all handler if it exists
	return append(params, rest), n.catchAllHandler
}
