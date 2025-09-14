package model

import (
	"time"
)

const (
	Term3months Term = "3m"
	Term1year   Term = "1y"
	Term2years  Term = "2y"
	Term3years  Term = "3y"
	Term4years  Term = "4y"
	Term5years  Term = "5y"
	Term6years  Term = "6y"
	Term7years  Term = "7y"
	Term8years  Term = "8y"
	Term9years  Term = "9y"
	Term10years Term = "10y"

	TypeListRate        Type = "listRate"
	TypeAverageRate     Type = "averageRate"
	TypeRatioDiscounted Type = "ratioDiscountedRate"
	TypeUnionDiscounted Type = "unionDiscountedRate"
)

type (
	Term string
	Type string
	Bank string
)

type RatioDiscountBoundary struct {
	MinRatio float32
	MaxRatio float32
}

type InterestSet struct {
	Bank          Bank
	Type          Type
	Term          Term
	NominalRate   float32
	ChangedOn     *time.Time
	LastCrawledAt time.Time

	RatioDiscountBoundaries *RatioDiscountBoundary // only for type ratioDiscounted
	UnionDiscount           bool                   // only true for type unionDiscounted
	AverageReferenceMonth   *AvgMonth              // only for type average
}

type AvgMonth struct {
	Month time.Month
	Year  uint
}
