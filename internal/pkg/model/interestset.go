package model

import (
	"time"
)

const (
	Term3months Term = "3m"
	Term6months Term = "6m"
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
	MinRatio float32 `json:"minRatio"`
	MaxRatio float32 `json:"maxRatio"`
}

type InterestSet struct {
	Bank          Bank       `json:"bank"`
	Type          Type       `json:"type"`
	Term          Term       `json:"term"`
	NominalRate   float32    `json:"nominalRate"`
	ChangedOn     *time.Time `json:"changedOn"`
	LastCrawledAt time.Time  `json:"lastCrawledAt"`

	RatioDiscountBoundaries *RatioDiscountBoundary `json:"ratioDiscountBoundaries"` // only for type ratioDiscounted
	UnionDiscount           bool                   `json:"unionDiscount"`           // only for type unionDiscounted
	AverageReferenceMonth   *AvgMonth              `json:"averageReferenceMonth"`   // only for type averageRate
}

type AvgMonth struct {
	Month time.Month `json:"month"`
	Year  uint       `json:"year"`
}
