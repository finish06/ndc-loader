package model

import "time"

// Application represents a Drugs@FDA application record.
type Application struct {
	ApplNo               string
	ApplType             *string
	SponsorName          *string
	MostRecentSubmission *time.Time
}

// DrugsFDAProduct represents a Drugs@FDA product record.
type DrugsFDAProduct struct {
	ID                int
	ApplNo            string
	ProductNo         string
	Form              *string
	Strength          *string
	ReferenceDrug     *string
	DrugName          *string
	ActiveIngredient  *string
	ReferenceStandard *string
}

// Submission represents a Drugs@FDA submission record.
type Submission struct {
	ID                             int
	ApplNo                         string
	SubmissionType                 *string
	SubmissionNo                   *string
	SubmissionStatus               *string
	SubmissionStatusDate           *time.Time
	SubmissionClassCode            *string
	SubmissionClassCodeDescription *string
}

// MarketingStatus represents a Drugs@FDA marketing status record.
type MarketingStatus struct {
	ID                int
	ApplNo            string
	ProductNo         *string
	MarketingStatusID *string
	Status            *string
}

// ActiveIngredient represents a Drugs@FDA active ingredient record.
type ActiveIngredient struct {
	ID             int
	ApplNo         string
	ProductNo      *string
	IngredientName *string
	Strength       *string
}

// TECode represents a Drugs@FDA therapeutic equivalence code record.
type TECode struct {
	ID        int
	ApplNo    string
	ProductNo *string
	TECode    *string
}
