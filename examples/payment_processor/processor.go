package main

import "context"

// PaymentResult is returned from processing.
type PaymentResult struct {
	TransactionID string
	Approved      bool
}

// PaymentProcessor is the interface shimmy will wrap.
type PaymentProcessor interface {
	Charge(ctx context.Context, accountID string, cents int) (PaymentResult, error)
}
