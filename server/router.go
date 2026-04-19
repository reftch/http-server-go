package server

// Router interface
type Router interface {
	GET(path string, handler HandlerFunc)
	POST(path string, handler HandlerFunc)
	Match(method, path string) (HandlerFunc, bool)
	// Add more methods as needed
}

// RouteHandler holds route and its handler
type RouteHandler struct {
	method  string
	path    string
	handler HandlerFunc
}

// SimpleRouter implementation
type SimpleRouter struct {
	handlers []RouteHandler
}

// NewSimpleRouter creates a new simple router
func NewSimpleRouter() *SimpleRouter {
	return &SimpleRouter{}
}

func (r *SimpleRouter) GET(path string, handler HandlerFunc) {
	r.handlers = append(r.handlers, RouteHandler{method: "GET", path: path, handler: handler})
}

func (r *SimpleRouter) POST(path string, handler HandlerFunc) {
	r.handlers = append(r.handlers, RouteHandler{method: "POST", path: path, handler: handler})
}

// Match finds the handler for the request
func (r *SimpleRouter) Match(method, path string) (HandlerFunc, bool) {
	for _, h := range r.handlers {
		if h.method == method && h.path == path {
			return h.handler, true
		}
	}
	return nil, false
}
