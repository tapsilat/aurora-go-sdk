package aurora

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// Field limits Aurora applies to an outcome report.
const (
	maxOutcomeExternalIDLen = 128
	maxOutcomeReasonCodeLen = 32
	maxOutcomeNoteLen       = 1000
)

// Values for OutcomeReportRequest.Source: who says what the transaction
// turned out to be. Aurora rejects OutcomeSourceOperator from an API token with
// ErrorCodeForbiddenSource; it is for panel users.
const (
	OutcomeSourceChargeback     = "chargeback"
	OutcomeSourceBankReport     = "bank_report"
	OutcomeSourcePartnerReport  = "partner_report"
	OutcomeSourceCustomerReport = "customer_report"
	OutcomeSourceOperator       = "operator"
)

// validOutcomeSources is the set of OutcomeReportRequest.Source values the
// SDK sends. Aurora reserves "review_case" for its own review decisions.
var validOutcomeSources = map[string]bool{
	OutcomeSourceChargeback:     true,
	OutcomeSourceBankReport:     true,
	OutcomeSourcePartnerReport:  true,
	OutcomeSourceCustomerReport: true,
	OutcomeSourceOperator:       true,
}

// Values for OutcomeReportRequest.Status.
const (
	OutcomeStatusActive    = "active"
	OutcomeStatusRetracted = "retracted"
)

// Values for OutcomeReportRequest.Outcome and OutcomeLabel.Label.
const (
	OutcomeFraud      = "fraud"
	OutcomeLegitimate = "legitimate"
)

// Values for OutcomeReportRequest.FraudType and OutcomeLabel.FraudType.
const (
	FraudTypeStolenCard      = "stolen_card"
	FraudTypeAccountTakeover = "account_takeover"
	FraudTypeFriendlyFraud   = "friendly_fraud"
	FraudTypeScam            = "scam"
	FraudTypeMerchantFraud   = "merchant_fraud"
	FraudTypeOther           = "other"
)

// validFraudTypes is the set of FraudType values Aurora accepts. Keep it in
// sync with the constants above.
var validFraudTypes = map[string]bool{
	FraudTypeStolenCard:      true,
	FraudTypeAccountTakeover: true,
	FraudTypeFriendlyFraud:   true,
	FraudTypeScam:            true,
	FraudTypeMerchantFraud:   true,
	FraudTypeOther:           true,
}

// Values for OutcomeReportResponse.Resolution: how Aurora found the
// transaction.
const (
	// ResolutionExact: the given ID matched one transaction.
	ResolutionExact = "exact"
	// ResolutionSuccessStatus: the payment ID matched several transactions and
	// the only successful one was taken.
	ResolutionSuccessStatus = "success_status"
	// ResolutionDuplicateCollapsed: the payment ID matched identical
	// duplicates and the oldest was taken.
	ResolutionDuplicateCollapsed = "duplicate_collapsed"
	// ResolutionLifecycleRoot: the match was a later step of a lifecycle and
	// the report was moved to its first transaction.
	ResolutionLifecycleRoot = "lifecycle_root"
)

// Values for OutcomeLabel.Tier.
const (
	// LabelTierExternal: the label comes from an outside source such as a
	// chargeback.
	LabelTierExternal = "external"
	// LabelTierInternal: the label comes only from a review decision.
	LabelTierInternal = "internal"
)

// OutcomeService tells Aurora what a transaction turned out to be after the
// decision, such as a chargeback that proves it was fraud.
type OutcomeService struct {
	client *Client
}

// OutcomeTransactionRef names the transaction a report is about. Set exactly
// one of the two fields.
type OutcomeTransactionRef struct {
	// TransactionID is Aurora's ID, as returned in ProcessResponse.
	TransactionID string `json:"transaction_id,omitempty"`
	// PaymentID is the common.payment_id sent when the transaction was
	// processed.
	PaymentID string `json:"payment_id,omitempty"`
}

// OutcomeReportRequest is the input to OutcomeService.Report.
type OutcomeReportRequest struct {
	// Source is one of the OutcomeSource constants. Required.
	Source string `json:"source"`
	// ExternalID is your ID for this report, such as the chargeback ID. With
	// Source it identifies the report: sending it again updates the same
	// report. Required, at most 128 characters.
	ExternalID string `json:"external_id"`
	// Version orders updates of one report. Aurora ignores a report whose
	// version is not higher than the stored one. Zero sends nothing and
	// Aurora uses 1.
	Version int64 `json:"version,omitempty"`
	// Status is OutcomeStatusActive (the default when empty) or
	// OutcomeStatusRetracted, which withdraws the report.
	Status string `json:"status,omitempty"`
	// Transaction names the transaction. Required.
	Transaction OutcomeTransactionRef `json:"transaction"`
	// Outcome is OutcomeFraud or OutcomeLegitimate. Required unless Status is
	// OutcomeStatusRetracted.
	Outcome string `json:"outcome,omitempty"`
	// FraudType is one of the FraudType constants. Required when Outcome is
	// OutcomeFraud.
	FraudType string `json:"fraud_type,omitempty"`
	// ReasonCode is the source's own reason code, at most 32 characters.
	ReasonCode string `json:"reason_code,omitempty"`
	// Amount is the fraud amount as a decimal string, such as "125.50". For
	// fraud it defaults to the transaction amount, and Aurora cuts it down to
	// the transaction amount when it is higher.
	Amount string `json:"amount,omitempty"`
	// Currency is an ISO 4217 code. It defaults to the transaction currency
	// and must match it.
	Currency string `json:"currency,omitempty"`
	// MoneyMoved tells whether the money actually left. Nil lets Aurora
	// decide: true for chargeback and bank_report, false otherwise.
	MoneyMoved *bool `json:"money_moved,omitempty"`
	// ReportedAt is when the source reported it. Zero means now.
	ReportedAt time.Time `json:"reported_at,omitzero"`
	// Note is free text, at most 1000 characters.
	Note string `json:"note,omitempty"`
}

// OutcomeLabel is the single label Aurora settles on for a transaction from
// all its active reports.
type OutcomeLabel struct {
	// Label is OutcomeFraud or OutcomeLegitimate.
	Label string `json:"label"`
	// Tier is LabelTierExternal or LabelTierInternal.
	Tier        string `json:"tier"`
	FraudType   string `json:"fraud_type,omitempty"`
	FraudAmount string `json:"fraud_amount,omitempty"`
	Currency    string `json:"currency,omitempty"`
	// Conflicting means active reports disagree; the label follows the
	// strongest source.
	Conflicting bool `json:"conflicting"`
	// Sources lists the sources of the active reports.
	Sources []string `json:"sources,omitempty"`
}

// OutcomeReportResponse is what Aurora returns for a report.
type OutcomeReportResponse struct {
	ReportID string `json:"report_id"`
	// TransactionID is the transaction the report was recorded on. It can
	// differ from the one sent; see Resolution.
	TransactionID string `json:"transaction_id"`
	// Created means a new report was stored (status 201).
	Created bool `json:"created"`
	// Changed means the report changed. False when the version was not
	// higher than the stored one.
	Changed bool `json:"changed"`
	// AmountClamped means Amount was higher than the transaction amount and
	// was cut down to it.
	AmountClamped bool `json:"amount_clamped"`
	// Resolution is one of the Resolution constants.
	Resolution string `json:"resolution"`
	// Label is the transaction's label after this report. Nil when no active
	// report is left.
	Label *OutcomeLabel `json:"label"`
}

// Report records what a transaction turned out to be (POST /outcomes). To
// withdraw a report, send it again with Status OutcomeStatusRetracted and a
// higher Version.
//
// Errors Aurora returns can be matched with errors.Is:
//   - ErrTransactionNotFound: no transaction matches.
//   - ErrAmbiguousTransaction: the payment ID matches several transactions;
//     APIError.CandidateCount says how many. Send the TransactionID instead.
//   - ErrExternalIDConflict: the external ID was used for another transaction.
//
// Other failures carry APIError.Code, such as ErrorCodeForbiddenSource.
//
// A request that fails validation returns a *ValidationError without
// contacting Aurora.
func (s *OutcomeService) Report(ctx context.Context, req *OutcomeReportRequest) (*OutcomeReportResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	httpReq, err := s.client.newRequest(ctx, http.MethodPost, pathOutcomes, req)
	if err != nil {
		return nil, err
	}

	var resp OutcomeReportResponse
	if err := s.client.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// validate applies Aurora's own checks before sending, so a request Aurora
// would reject fails without a network call.
func (req *OutcomeReportRequest) validate() error {
	if req == nil {
		return &ValidationError{Field: "request", Message: "request is required"}
	}
	if !validOutcomeSources[req.Source] {
		return &ValidationError{Field: "source", Message: "invalid or missing source"}
	}
	if strings.TrimSpace(req.ExternalID) == "" {
		return &ValidationError{Field: "external_id", Message: "external_id is required"}
	}
	if utf8.RuneCountInString(req.ExternalID) > maxOutcomeExternalIDLen {
		return &ValidationError{Field: "external_id", Message: fmt.Sprintf("external_id is longer than %d characters", maxOutcomeExternalIDLen)}
	}
	if req.Version < 0 {
		return &ValidationError{Field: "version", Message: "version must be 1 or higher"}
	}
	if req.Status != "" && req.Status != OutcomeStatusActive && req.Status != OutcomeStatusRetracted {
		return &ValidationError{Field: "status", Message: "status must be active or retracted"}
	}

	// A whitespace-only value would still be sent, so reject it on its own.
	hasID := req.Transaction.TransactionID != ""
	hasPayment := req.Transaction.PaymentID != ""
	if hasID && strings.TrimSpace(req.Transaction.TransactionID) == "" {
		return &ValidationError{Field: "transaction.transaction_id", Message: "transaction_id is blank"}
	}
	if hasPayment && strings.TrimSpace(req.Transaction.PaymentID) == "" {
		return &ValidationError{Field: "transaction.payment_id", Message: "payment_id is blank"}
	}
	if hasID == hasPayment {
		return &ValidationError{Field: "transaction", Message: "set exactly one of transaction_id and payment_id"}
	}

	active := req.Status != OutcomeStatusRetracted
	switch {
	case req.Outcome == "" && active:
		return &ValidationError{Field: "outcome", Message: "outcome is required"}
	case req.Outcome != "" && req.Outcome != OutcomeFraud && req.Outcome != OutcomeLegitimate:
		return &ValidationError{Field: "outcome", Message: "outcome must be fraud or legitimate"}
	}
	switch {
	case req.FraudType == "" && req.Outcome == OutcomeFraud && active:
		return &ValidationError{Field: "fraud_type", Message: "fraud_type is required for fraud"}
	case req.FraudType != "" && !validFraudTypes[req.FraudType]:
		return &ValidationError{Field: "fraud_type", Message: "unknown fraud_type"}
	}

	if utf8.RuneCountInString(req.ReasonCode) > maxOutcomeReasonCodeLen {
		return &ValidationError{Field: "reason_code", Message: fmt.Sprintf("reason_code is longer than %d characters", maxOutcomeReasonCodeLen)}
	}
	if utf8.RuneCountInString(req.Note) > maxOutcomeNoteLen {
		return &ValidationError{Field: "note", Message: fmt.Sprintf("note is longer than %d characters", maxOutcomeNoteLen)}
	}
	return nil
}
