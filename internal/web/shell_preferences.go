package web

import "context"

type sidebarWidthKey struct{}

// WithSidebarWidth carries a validated presentation preference to every page
// rendered through PortalDocument, independent of its tenant or shell.
func WithSidebarWidth(ctx context.Context, width int) context.Context {
	if width < 240 || width > 420 {
		width = 280
	}
	return context.WithValue(ctx, sidebarWidthKey{}, width)
}

func portalSidebarWidth(ctx context.Context) int {
	if width, ok := ctx.Value(sidebarWidthKey{}).(int); ok {
		return width
	}
	return 280
}
