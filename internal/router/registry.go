package router

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	permissionPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*:[a-z][a-z0-9_]*$`)
	auditPattern      = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`)
)

var allowedMethods = map[string]struct{}{
	"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "HEAD": {}, "OPTIONS": {},
}

// Route is the contract for one HTTP operation. Registration is not allowed to
// be implicit: every mutating route must declare an audit action.
type Route struct {
	Method        string
	Path          string
	OperationID   string
	Summary       string
	Tags          []string
	Permission    string
	AuditAction   string
	Public        bool
	DefaultStatus int
}

// Registry is the single source of truth for route metadata. It is populated
// during API registration and validated before the server starts.
type Registry struct {
	mu     sync.RWMutex
	routes []Route
	keys   map[string]struct{}
	ops    map[string]struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		keys: make(map[string]struct{}),
		ops:  make(map[string]struct{}),
	}
}

func (r *Registry) Add(route Route) error {
	route.Method = strings.ToUpper(strings.TrimSpace(route.Method))
	route.Path = normalizePath(route.Path)
	route.OperationID = strings.TrimSpace(route.OperationID)

	if _, ok := allowedMethods[route.Method]; !ok {
		return fmt.Errorf("route %q: unsupported method %q", route.Path, route.Method)
	}
	if route.Path == "" || !strings.HasPrefix(route.Path, "/") {
		return fmt.Errorf("route %q %q: path must start with /", route.Method, route.Path)
	}
	if route.OperationID == "" {
		return fmt.Errorf("route %s %s: operation id is required", route.Method, route.Path)
	}
	if route.Public && route.Permission != "" {
		return fmt.Errorf("route %s %s: public route cannot require permission", route.Method, route.Path)
	}
	if route.Permission != "" && !permissionPattern.MatchString(route.Permission) {
		return fmt.Errorf("route %s %s: invalid permission %q (want domain:action)", route.Method, route.Path, route.Permission)
	}
	if route.AuditAction != "" && !auditPattern.MatchString(route.AuditAction) {
		return fmt.Errorf("route %s %s: invalid audit action %q (want domain.verb)", route.Method, route.Path, route.AuditAction)
	}
	if requiresAudit(route.Method) && route.AuditAction == "" {
		return fmt.Errorf("route %s %s: mutating route must declare an audit action", route.Method, route.Path)
	}
	if route.DefaultStatus != 0 && (route.DefaultStatus < 200 || route.DefaultStatus > 599) {
		return fmt.Errorf("route %s %s: invalid default status %d", route.Method, route.Path, route.DefaultStatus)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := route.Method + " " + route.Path
	if _, ok := r.keys[key]; ok {
		return fmt.Errorf("duplicate route %s", key)
	}
	if _, ok := r.ops[route.OperationID]; ok {
		return fmt.Errorf("duplicate operation id %q", route.OperationID)
	}
	r.keys[key] = struct{}{}
	r.ops[route.OperationID] = struct{}{}
	route.Tags = append([]string(nil), route.Tags...)
	r.routes = append(r.routes, route)
	return nil
}

// Validate is the startup gate. Add already rejects bad routes, but Validate
// makes the invariant explicit and catches an empty registry.
func (r *Registry) Validate() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.routes) == 0 {
		return errors.New("no routes registered")
	}
	for _, route := range r.routes {
		if requiresAudit(route.Method) && route.AuditAction == "" {
			return fmt.Errorf("route %s %s: missing audit action", route.Method, route.Path)
		}
	}
	return nil
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.routes)
}

func (r *Registry) Routes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Route, len(r.routes))
	copy(out, r.routes)
	return out
}

// Has reports whether an exact method+path pattern is registered.
func (r *Registry) Has(method, path string) bool {
	method = strings.ToUpper(method)
	path = normalizePath(path)

	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.keys[method+" "+path]
	return ok
}

// IsPublic reports whether a request path matches a public route pattern.
func (r *Registry) IsPublic(method, path string) bool {
	method = strings.ToUpper(method)
	path = normalizePath(path)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, route := range r.routes {
		if route.Public && route.Method == method && matchPath(route.Path, path) {
			return true
		}
	}
	return false
}

// AuditAction returns the declared action for a concrete request path.
func (r *Registry) AuditAction(method, path string) string {
	method = strings.ToUpper(method)
	path = normalizePath(path)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, route := range r.routes {
		if route.Method == method && matchPath(route.Path, path) {
			return route.AuditAction
		}
	}
	return ""
}

func requiresAudit(method string) bool {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return false
	default:
		return true
	}
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if path != "/" {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for i := range patternParts {
		part := patternParts[i]
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}
		if part != pathParts[i] {
			return false
		}
	}
	return true
}
