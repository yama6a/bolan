// Package store provides data persistence interfaces and implementations.
//
//go:generate go run -mod=mod github.com/matryer/moq -out storemock/store_mock.go -pkg storemock . Store
package store

import "github.com/yama6a/bolan-compare/internal/pkg/model"

// Store defines the interface for persisting interest rate data.
type Store interface {
	UpsertInterestSet(set model.InterestSet) error
	GetInterestSets() ([]model.InterestSet, error)
}
