package routix

import (
	"net"
	"net/http"
)

type routingState struct {
	get        *methodRoutes
	methods    map[string]*methodRoutes
	middleware []Middleware
	notFound   Handler
	notMethod  Handler
	proxies    []*net.IPNet
}

type methodRoutes struct {
	root     *node
	static   map[string]*node
	prefixes *routePrefix
}

func (r *Router) routingState() *routingState {
	if state := r.dispatch.Load(); state != nil {
		return state
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if state := r.dispatch.Load(); state != nil {
		return state
	}
	state := &routingState{
		methods:    make(map[string]*methodRoutes, len(r.trees)),
		middleware: append([]Middleware(nil), r.middleware...),
		notFound:   r.notFound,
		notMethod:  r.notMethod,
		proxies:    r.trustedProxies,
	}
	for method, root := range r.trees {
		routes := &methodRoutes{}
		routes.root = routes.copyNode(root, "", false)
		if routes.prefixes != nil {
			routes.prefixes.flatten()
		}
		state.methods[method] = routes
		if method == http.MethodGet {
			state.get = routes
		}
	}
	r.dispatch.Store(state)
	return state
}

func (m *methodRoutes) copyNode(source *node, prefix string, dynamic bool) *node {
	if source == nil {
		return nil
	}
	cloned := *source
	if len(cloned.paramNames) == 1 {
		cloned.paramName = cloned.paramNames[0]
	}
	cloned.children = nil
	if len(source.children) != 0 {
		cloned.children = make(map[string]*node, len(source.children))
		for key, child := range source.children {
			cloned.children[key] = m.copyNode(child, prefix+"/"+key, dynamic || key == ":" || key == "*")
		}
	}
	cloned.paramChild = cloned.children[":"]
	cloned.wildcardChild = cloned.children["*"]
	if cloned.handler != nil && !dynamic {
		if m.static == nil {
			m.static = make(map[string]*node)
		}
		path := prefix
		if path == "" {
			path = "/"
		}
		m.static[path] = &cloned
	}
	if !dynamic && onlyDynamicChildren(cloned.children) {
		if child := cloned.paramChild; child != nil && child.handler != nil {
			m.addPrefix(prefix+"/", child, false)
		}
		if child := cloned.wildcardChild; child != nil && child.handler != nil && cloned.paramChild == nil && cloned.handler == nil {
			m.addPrefix(prefix+"/", child, true)
		}
	}
	return &cloned
}

func onlyDynamicChildren(children map[string]*node) bool {
	for key := range children {
		if key != ":" && key != "*" {
			return false
		}
	}
	return true
}

func (s *routingState) routesFor(method string) *methodRoutes {
	if method == http.MethodGet {
		return s.get
	}
	return s.methods[method]
}
