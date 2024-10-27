package crawler

import (
	"errors"
	"fmt"
)

var (
	ErrNoInterestSetFound       = errors.New("no interest set found")
	ErrUnsupportedTerm          = errors.New("unsupported term")
	ErrUnsupportedInterestRate  = errors.New("unsupported interest rate")
	ErrUnsupportedChangeDate    = errors.New("unsupported change date")
	ErrUnsupportedAvgMonth      = errors.New("unsupported avg month")
	ErrUnsupportedReferenceDate = fmt.Errorf("unsupported reference date format")
)
