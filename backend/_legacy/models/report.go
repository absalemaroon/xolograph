package models

import "time"

type CaseReport struct {
	GeneratedAt       time.Time       `json:"generatedAt"`
	Investigation     Investigation   `json:"investigation"`
	EvidenceItems     []EvidenceItem  `json:"evidenceItems"`
	AuditEvents       []AuditEvent    `json:"auditEvents"`
	Entities          []Entity        `json:"entities"`
	TotalVaultVolume  int64           `json:"totalVaultVolume"`
	IntegrityStatus   string          `json:"integrityStatus"`
	ReportChecksum    string          `json:"reportChecksum"`
}