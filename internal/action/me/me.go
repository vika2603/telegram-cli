// Package me contains the action for printing the current account identity.
package me

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// FetchFunc loads the current account identity.
type FetchFunc func(context.Context) (output.UserRow, error)

// Show delegates identity loading after wiring validation.
func Show(ctx context.Context, fetch FetchFunc) (output.UserRow, error) {
	if fetch == nil {
		return output.UserRow{}, fmt.Errorf("%w: me called without fetch function", command.ErrPrecondition)
	}
	return fetch(ctx)
}
