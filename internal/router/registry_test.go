package router

import "testing"

func TestAddRejectsMutatingRouteWithoutAudit(t *testing.T) {
	reg := NewRegistry()
	err := reg.Add(Route{Method: "POST", Path: "/api/admin/v1/articles", OperationID: "article.create"})
	if err == nil {
		t.Fatal("expected an error for a mutating route without an audit action")
	}
}

func TestAddRejectsPublicRouteWithPermission(t *testing.T) {
	reg := NewRegistry()
	err := reg.Add(Route{
		Method:      "GET",
		Path:        "/public",
		OperationID: "public",
		Public:      true,
		Permission:  "article:read",
	})
	if err == nil {
		t.Fatal("expected an error for a public route with a permission")
	}
}

func TestAddRejectsDuplicateRoute(t *testing.T) {
	reg := NewRegistry()
	route := Route{Method: "GET", Path: "/healthz", OperationID: "health"}
	if err := reg.Add(route); err != nil {
		t.Fatalf("first add: %v", err)
	}
	route.OperationID = "health.duplicate"
	if err := reg.Add(route); err == nil {
		t.Fatal("expected duplicate route error")
	}
}

func TestIsPublicMatchesPathParameters(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Add(Route{
		Method:      "GET",
		Path:        "/articles/{id}",
		OperationID: "article.public",
		Public:      true,
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	if !reg.IsPublic("GET", "/articles/123") {
		t.Error("expected /articles/123 to be public")
	}
	if reg.IsPublic("GET", "/articles/123/comments") {
		t.Error("expected nested path not to match")
	}
	if reg.IsPublic("POST", "/articles/123") {
		t.Error("expected a different method not to match")
	}
}

func TestValidateRequiresRoutes(t *testing.T) {
	if err := NewRegistry().Validate(); err == nil {
		t.Fatal("expected an error for an empty registry")
	}
}
