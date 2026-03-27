package api

import (
	"context"
	"fmt"

	"github.com/calebdunn/ndc-loader/internal/store"
)

type mockQueryProvider struct {
	lookupProductFn func(ctx context.Context, variants []string) (*store.ProductResult, error)
	lookupPackageFn func(ctx context.Context, variants []string) (*store.ProductResult, string, error)
	searchFn        func(ctx context.Context, query string, limit, offset int) ([]store.SearchResult, int, error)
	packagesFn      func(ctx context.Context, productNDC string) ([]store.PackageResult, error)
	statsFn         func(ctx context.Context) (*store.StatsResult, error)
	openFDASearchFn func(ctx context.Context, where string, args []interface{}, limit, skip int) ([]store.ProductResult, int, error)
}

func (m *mockQueryProvider) LookupByProductNDC(ctx context.Context, variants []string) (*store.ProductResult, error) {
	if m.lookupProductFn != nil {
		return m.lookupProductFn(ctx, variants)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockQueryProvider) LookupByPackageNDC(ctx context.Context, variants []string) (*store.ProductResult, string, error) {
	if m.lookupPackageFn != nil {
		return m.lookupPackageFn(ctx, variants)
	}
	return nil, "", fmt.Errorf("not found")
}

func (m *mockQueryProvider) SearchProducts(ctx context.Context, query string, limit, offset int) ([]store.SearchResult, int, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, query, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockQueryProvider) GetPackagesByProductNDC(ctx context.Context, productNDC string) ([]store.PackageResult, error) {
	if m.packagesFn != nil {
		return m.packagesFn(ctx, productNDC)
	}
	return nil, nil
}

func (m *mockQueryProvider) GetStats(ctx context.Context) (*store.StatsResult, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx)
	}
	return &store.StatsResult{}, nil
}

func (m *mockQueryProvider) OpenFDASearch(ctx context.Context, where string, args []interface{}, limit, skip int) ([]store.ProductResult, int, error) {
	if m.openFDASearchFn != nil {
		return m.openFDASearchFn(ctx, where, args, limit, skip)
	}
	return nil, 0, nil
}

// Compile-time check.
var _ QueryProvider = (*mockQueryProvider)(nil)
