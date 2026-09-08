package repo

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
)

type transactionContextKey struct{}

// QueryFromContext resolves the local transaction; never mutate a shared service's query.
func QueryFromContext(ctx context.Context, fallback *query.Query) *query.Query {
	if tx, ok := ctx.Value(transactionContextKey{}).(*query.Query); ok {
		return tx
	}
	return fallback
}

// InTransaction joins the caller's local transaction. Callers must propagate errors
// and must not pass this context to goroutines or remote payment operations.
func InTransaction(ctx context.Context, q *query.Query, fn func(context.Context, *query.Query) error) error {
	if tx, ok := ctx.Value(transactionContextKey{}).(*query.Query); ok {
		return fn(ctx, tx)
	}
	return q.Transaction(func(tx *query.Query) error {
		return fn(context.WithValue(ctx, transactionContextKey{}, tx), tx)
	})
}
