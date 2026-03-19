package main

import (
	"context"
	"fmt"
)

func main() {
	inner := NewFlakyGateway()

	processor := &PaymentProcessorShim{
		Inner: inner,
		Middleware: []PaymentProcessorMiddleware{
			&RetryOnceMiddleware{},
			NewAuditingInterceptor(),
		},
	}

	ctx := context.Background()
	result, err := processor.Charge(ctx, "acct-42", 2599)
	if err != nil {
		fmt.Println("charge failed:", err)
		return
	}

	fmt.Printf("payment result: tx=%s approved=%v\n", result.TransactionID, result.Approved)
}
