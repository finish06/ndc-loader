package model

import "time"

// Product represents an NDC Directory product record.
type Product struct {
	ProductID          string
	ProductNDC         string
	ProductType        *string
	ProprietaryName    *string
	NonproprietaryName *string
	DosageForm         *string
	Route              *string
	LabelerName        *string
	SubstanceName      *string
	Strength           *string
	StrengthUnit       *string
	PharmClasses       *string
	DEASchedule        *string
	MarketingCategory  *string
	ApplicationNumber  *string
	MarketingStart     *time.Time
	MarketingEnd       *time.Time
	NDCExclude         bool
	ListingCertified   *time.Time
}

// Package represents an NDC Directory package record.
type Package struct {
	ID              int
	ProductID       string
	ProductNDC      string
	NDCPackageCode  string
	Description     *string
	MarketingStart  *time.Time
	MarketingEnd    *time.Time
	NDCExclude      bool
	SamplePackage   bool
}
