package router

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"

	"github.com/example/homebrew-api-starter/internal/auth"
)

// Register is the only way API modules add routes. It records the contract
// (permission + audit action) and installs the authorization check in one
// place, so a route cannot silently skip either.
func Register[I, O any](
	reg *Registry,
	api huma.API,
	route Route,
	handler func(context.Context, *I) (*O, error),
) (err error) {
	if err := reg.Add(route); err != nil {
		return err
	}

	op := huma.Operation{
		OperationID: route.OperationID,
		Method:      route.Method,
		Path:        route.Path,
		Summary:     route.Summary,
		Tags:        route.Tags,
	}
	if route.DefaultStatus != 0 {
		op.DefaultStatus = route.DefaultStatus
	}

	// huma.Register panics on invalid or duplicate operations. Convert that to
	// an ordinary startup error so the process fails with a useful message.
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("register %s %s: %v", route.Method, route.Path, rec)
		}
	}()

	huma.Register(api, op, func(ctx context.Context, in *I) (*O, error) {
		if route.Permission != "" {
			principal, ok := auth.PrincipalFrom(ctx)
			if !ok {
				return nil, problem(401, "Unauthorized", "authentication required")
			}
			if !principal.Has(route.Permission) {
				return nil, problem(403, "Forbidden", "missing permission: "+route.Permission)
			}
		}
		return handler(ctx, in)
	})
	return nil
}

func problem(status int, title, detail string) *huma.ErrorModel {
	return &huma.ErrorModel{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	}
}
