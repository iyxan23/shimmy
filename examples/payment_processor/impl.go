package main

import (
	"context"
	"errors"
	"fmt"
)

// FlakyGateway simulates one transient failure per account.
type FlakyGateway struct {
	failedOnce map[string]bool
	counter    int
}

func NewFlakyGateway() *FlakyGateway {
	return &FlakyGateway{failedOnce: map[string]bool{}}
}

func (g *FlakyGateway) Charge(_ context.Context, accountID string, cents int) (PaymentResult, error) {
	if cents <= 0 {
		return PaymentResult{}, errors.New("amount must be positive")
	}

	if !g.failedOnce[accountID] {
		g.failedOnce[accountID] = true
		return PaymentResult{}, errors.New("temporary gateway timeout")
	}

	g.counter++
	return PaymentResult{
		TransactionID: fmt.Sprintf("tx-%d", g.counter),
		Approved:      true,
	}, nil
}
