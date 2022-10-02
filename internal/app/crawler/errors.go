package crawler

import "errors"

var (
	ErrNoInterestSetFound      = errors.New("no interest set found")
	ErrUnsupportedTerm         = errors.New("unsupported term")
	ErrUnsupportedInterestRate = errors.New("unsupported interest rate")
	ErrUnsupportedChangeDate   = errors.New("unsupported change date")
	ErrUnsupportedAvgMonth     = errors.New("unsupported avg month")
)
