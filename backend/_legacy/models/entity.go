package models

import "time"

type AttributionSource string

const (
	SourceVerified    AttributionSource = "VERIFIED_SOURCE"
	SourceThirdParty  AttributionSource = "THIRD_PARTY_PROVIDER"
	SourceAnalyst     AttributionSource = "ANALYST_ASSERTION"
	SourceAlgorithmic AttributionSource = "ALGORITHMIC_INFERENCE"
	SourceUnknown     AttributionSource = "UNKNOWN"
)

type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "HIGH"
	ConfidenceMedium ConfidenceLevel = "MEDIUM"
	ConfidenceLow    ConfidenceLevel = "LOW"
)

type Entity struct {
	ID                 string            `json:"id"`
	InvestigationID    string            `json:"investigationId"`
	Name               string            `json:"name"`
	Category           string            `json:"category"`
	AttributionSource  AttributionSource `json:"attributionSource"`
	Confidence         ConfidenceLevel   `json:"confidence"`
	SupportingEvidence string            `json:"supportingEvidence"`
	AnalystNotes       string            `json:"analystNotes"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
	Addresses          []EntityAddress   `json:"addresses"`
}

type EntityAddress struct {
	EntityID     string    `json:"entityId"`
	Address      string    `json:"address"`
	Network      NetworkID `json:"network"`
	DateAttached time.Time `json:"dateAttached"`
	AttachedBy   string    `json:"attachedBy"`
}