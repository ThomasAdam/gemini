package gemini

import (
	"context"
	"fmt"
	"strings"
)

type node struct {
	// This reference to a given ServeMux should only be used for settings.
	mux *ServeMux

	// Storing a reference to the parent is used for finding the most relevant
	// catchAllHandler.
	parent *node

	handler         Handler
	slashHandler    Handler
	catchAllHandler Handler
	children        map[string]*node
	param           *node
}

func newNode(parent *node) *node {
	ret := &node{children: make(map[string]*node), parent: parent}
	if parent != nil {
		ret.mux = parent.mux
	}
	return ret
}

func (n *node) ServeGemini(ctx context.Context, w ResponseWriter, r *Request) {
	params, handler := n.match(r.URL.Path, n.mux.RedirectSlash)
	if handler == nil {
		return
	}

	ctx = CtxWithParams(ctx, params)
	handler.ServeGemini(ctx, w, r)
}

func (n *node) Handle(pattern string, h Handler) {
	pattern = cleanPath(pattern)
	hasRest := strings.HasSuffix(pattern, "/:rest")
	hasSlash := strings.HasSuffix(pattern, "/")

	target := n.ensureNode(pattern)

	if hasRest {
		if target.catchAllHandler != nil {
			panic("overlapping catchAllHandlers")
		}
		target.catchAllHandler = h
	} else if hasSlash {
		if target.slashHandler != nil {
			panic("overlapping handlers")
		}
		target.slashHandler = h
	} else {
		if target.handler != nil {
			panic("overlapping handlers")
		}
		target.handler = h
	}
}

func (n *node) NotFound(h Handler) {
	target := n.ensureNode(":rest")
	if target.catchAllHandler != nil {
		panic("overlapping catchAllHandlers")
	}
	target.catchAllHandler = h
}

func (n *node) Route(pattern string, fn func(r Router)) Router {
	target := n.ensureNode(cleanPath(pattern))
	fn(target)
	return target
}

func (n *node) ensureNode(targetPath string) *node {
	// NOTE: this assumes a pre-cleaned path has been passed in. ALL CALLERS
	// MUST USE cleanPath BEFORE CALLING THIS FUNCTION.
	targetPath = strings.Trim(targetPath, "/")
	return n.ensureNodeImpl(targetPath)
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
			n.param = newNode(n)
		}

		return n.param.ensureNodeImpl(rest)
	}

	target := n.children[next]
	if target == nil {
		n.children[next] = newNode(n)
		target = n.children[next]
	}

	return target.ensureNodeImpl(rest)
}

func (n *node) match(targetPath string, allowRedirect bool) ([]string, Handler) {
	targetPath = strings.TrimPrefix(cleanPath(targetPath), "/")
	hasSlash := strings.HasSuffix(targetPath, "/")
	return n.matchImpl(targetPath, targetPath, allowRedirect, hasSlash, nil)
}

func (n *node) matchImpl(origPath string, path string, allowRedirect bool, hasSlash bool, params []string) ([]string, Handler) {
	if n == nil {
		return nil, nil
	}

	if path == "" {
		if hasSlash {
			if n.slashHandler != nil {
				return params, n.slashHandler
			}

			if allowRedirect && n.handler != nil {
				return params, HandlerFunc(redirectRemoveSlash)
			}
		} else {
			if n.handler != nil {
				return params, n.handler
			}

			if allowRedirect && n.slashHandler != nil {
				return params, HandlerFunc(redirectAddSlash)
			}
		}

		return params, n.catchAllHandler
	}

	next, rest := pathSegment(path)

	// First attempt static routes.
	retParams, retHandler := n.children[next].matchImpl(origPath, rest, allowRedirect, hasSlash, params)
	if retHandler != nil {
		return retParams, retHandler
	}

	// If there isn't a matching static route, attempt a param route.
	retParams, retHandler = n.param.matchImpl(origPath, rest, allowRedirect, hasSlash, append(params, next))
	if retHandler != nil {
		return retParams, retHandler
	}

	// Finally fall back to the catch all handler if it exists. Note that we
	// also redirect to include a slash because all catchAllHandlers should
	// match after a path separator. This fixes a number of edge cases with the
	// gemini.FileServer when using it with gemini.StripPrefix.
	if allowRedirect && !hasSlash {
		return params, HandlerFunc(redirectAddSlash)
	}

	// Traverse back up the tree to find the most relevant catchAllHandler.
	target := n
	handler := n.catchAllHandler
	for handler == nil && target.parent != nil {
		target = target.parent
		handler = target.catchAllHandler
	}

	return append(params, rest), handler
}

func redirectAddSlash(ctx context.Context, w ResponseWriter, r *Request) {
	w.WriteStatus(StatusRedirect, cleanPath(r.URL.Path)+"/")
}

func redirectRemoveSlash(ctx context.Context, w ResponseWriter, r *Request) {
	w.WriteStatus(StatusRedirect, strings.TrimSuffix(cleanPath(r.URL.Path), "/"))
}

// Print is used for debugging the tree
func (n *node) print(prefix string) {
	//fmt.Println("Node at", prefix)
	//fmt.Println(n.children)

	if n.catchAllHandler != nil {
		fmt.Println(prefix, "catchAll")
	}

	if n.handler != nil {
		fmt.Println(prefix)
	}

	if n.slashHandler != nil {
		fmt.Println(prefix + "/")
	}

	for k, v := range n.children {
		v.print(prefix + "/" + k)
	}

	if n.param != nil {
		n.param.print(prefix + "/:param")
	}
}
