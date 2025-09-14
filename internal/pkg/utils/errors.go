package utils

import (
	"errors"
)

var (
	ErrNoInterestSetFound      = errors.New("no interest set found")
	ErrUnsupportedTerm         = errors.New("unsupported term")
	ErrTermHeader              = errors.New("row is a term header row")
	ErrUnsupportedInterestRate = errors.New("unsupported interest rate")
	ErrUnsupportedChangeDate   = errors.New("unsupported change date")
	ErrUnsupportedAvgMonth     = errors.New("unsupported avg month")
)
