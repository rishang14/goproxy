package httpx

type Router struct {
	routes map[string]map[string]Handler
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]map[string]Handler)}
}

func (rt *Router) Handle(method string, path string, h Handler) {
	if rt.routes[path] == nil {
		rt.routes[path] = make(map[string]Handler)
	}
	rt.routes[path][method] = h
}

func (rt *Router) ServeHTTP(w ResponseWriter, r *Request) {
	methods, ok := rt.routes[r.Path]
	if !ok {
		w.WriteResponse(404)
		w.WriteBody([]byte("not found"))
		return
	}

	handler, ok := methods[r.Method]
	if !ok {
		w.WriteResponse(405)
		w.WriteBody([]byte("method not allowed"))
		return
	}

	handler.ServeHTTP(w, r)
}
