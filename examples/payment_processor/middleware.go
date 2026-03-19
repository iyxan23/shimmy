package main

import (
	"context"
	"fmt"

	"shimmy"
)

// RetryOnceMiddleware retries Charge once on error.
type RetryOnceMiddleware struct {
	BasePaymentProcessorMiddleware
}

func (m *RetryOnceMiddleware) AroundCharge(
	ctx context.Context,
	accountID string,
	cents int,
	next func(context.Context, string, int) (PaymentResult, error),
) (PaymentResult, error) {
	res, err := next(ctx, accountID, cents)
	if err == nil {
		return res, nil
	}
	fmt.Printf("[typed-mw] first attempt failed, retrying: %v\n", err)
	return next(ctx, accountID, cents)
}

// NewAuditingInterceptor prints args and results for each call.
func NewAuditingInterceptor() PaymentProcessorMiddleware {
	return NewPaymentProcessorInterceptor(func(call shimmy.Call, invoke func()) {
		fmt.Printf("[interceptor] begin %s args=%v\n", call.Method(), call.Args())
		invoke()
		fmt.Printf("[interceptor] end %s results=%v\n", call.Method(), call.Results())
	})
}
