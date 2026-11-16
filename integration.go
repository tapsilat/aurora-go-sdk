package aurora

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// IntegrationService tells Aurora what an integration sends, so rules can be
// checked against it before any real transaction arrives.
type IntegrationService struct {
	client *Client
}

// Field names to list in IntegrationSample.FixedFields. Any transaction field
// name Aurora knows is accepted; these are the usual ones.
const (
	FieldTransactionType  = "transaction_type"
	FieldPaymentMethod    = "payment_method"
	FieldCurrency         = "currency"
	FieldTransferType     = "transfer_type"
	FieldWithdrawalMethod = "withdrawal_method"
	FieldAccountType      = "account_type"
	FieldDepositMethod    = "deposit_method"
	FieldIsRemoteDeposit  = "is_remote_deposit"
	FieldCategory         = "category"

	FieldAccountAgeDays          = "account_age_days"
	FieldCustomerAccountType     = "customer_account_type"
	FieldCustomerAgeYears        = "customer_age_years"
	FieldRegisteredIncome        = "registered_income"
	FieldIncomeMultiplier        = "income_multiplier"
	FieldIsFirstTransfer         = "is_first_transfer"
	FieldLastLoginTime           = "last_login_time"
	FieldPasswordChangedRecently = "password_changed_recently"
	FieldUserID                  = "user_id"
	FieldEmail                   = "email"
	FieldPhone                   = "phone"
	FieldIdentityNumber          = "identity_number"
	FieldIPAddress               = "ip_address"
	FieldBrowserAgent            = "browser_agent"
	FieldDeviceID                = "device_id"
	FieldBillingName             = "billing_name"
	FieldShippingAddress         = "shipping_address"
)

// Reasons a flow never carries a field, for IntegrationSample.Absent.
const (
	// AbsentNotAMember: the party is not a member of the platform, so there
	// is no account data (a partner institution sending money).
	AbsentNotAMember = "not_a_member"
	// AbsentNoSession: the transaction does not come from a user session, so
	// there is no IP or device (a payment matched from a bank statement).
	AbsentNoSession = "no_session"
	// AbsentNotAnOrder: the transaction is not tied to an order, so there is
	// no billing or shipping information.
	AbsentNotAnOrder = "not_an_order"
	// AbsentNotCollected: the flow does not collect it.
	AbsentNotCollected = "not_collected"
)

var validAbsentReasons = map[string]bool{
	AbsentNotAMember: true, AbsentNoSession: true, AbsentNotAnOrder: true, AbsentNotCollected: true,
}

// AbsentField is a field a flow never carries, and why. Aurora then tells a
// rule that needs it is not meant for this flow, instead of reporting
// missing data.
type AbsentField struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// IntegrationSample is one kind of transaction the integration sends, built
// the way it builds real ones.
type IntegrationSample struct {
	// Flow names the kind, e.g. "bank_withdraw"; Variant tells samples of one
	// flow apart, e.g. a currency. Flow and Variant together must be unique.
	Flow    string `json:"flow"`
	Variant string `json:"variant,omitempty"`
	// Title is shown to users, e.g. "Banka çekimi".
	Title string `json:"title,omitempty"`
	// FixedFields lists fields whose value is the same on every transaction
	// of this flow (a bank withdrawal always has payment_method
	// "bank_withdraw"). Other fields with a value are taken to vary, and
	// fields left empty are taken to be never sent. The transaction type is
	// always fixed.
	FixedFields []string `json:"fixed_fields,omitempty"`
	// Absent lists fields this flow never carries by its nature, with why.
	// They must be empty in Transaction.
	Absent []AbsentField `json:"absent,omitempty"`
	// Transaction is a sample as it would be sent to Process.Execute.
	Transaction *Transaction `json:"transaction"`
}

// PublishSamplesRequest replaces every sample Source published before.
type PublishSamplesRequest struct {
	// Source names the integration, e.g. "admin".
	Source  string              `json:"source"`
	Samples []IntegrationSample `json:"samples"`
}

// PublishSamplesResponse reports what Aurora stored.
type PublishSamplesResponse struct {
	Source string `json:"source"`
	Stored int    `json:"stored"`
}

// PublishSamples replaces the samples of req.Source for the organization the
// API key belongs to (PUT /integration/samples). Publishing an empty list
// removes them. Aurora's rule self-test then reports, for every rule, on which
// of these flows it can act and why not on the others.
func (s *IntegrationService) PublishSamples(ctx context.Context, req *PublishSamplesRequest) (*PublishSamplesResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	httpReq, err := s.client.newRequest(ctx, http.MethodPut, pathIntegrationSamples, req)
	if err != nil {
		return nil, err
	}

	var resp PublishSamplesResponse
	if err := s.client.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (req *PublishSamplesRequest) validate() error {
	if req == nil {
		return &ValidationError{Field: "request", Message: "request is required"}
	}
	if strings.TrimSpace(req.Source) == "" {
		return &ValidationError{Field: "source", Message: "source is required"}
	}
	seen := make(map[string]bool, len(req.Samples))
	for i, sample := range req.Samples {
		field := fmt.Sprintf("samples[%d]", i)
		if strings.TrimSpace(sample.Flow) == "" {
			return &ValidationError{Field: field + ".flow", Message: "flow is required"}
		}
		if sample.Transaction == nil || !validTransactionTypes[sample.Transaction.Type] {
			return &ValidationError{Field: field + ".transaction.type", Message: "invalid or missing transaction type"}
		}
		absent := make(map[string]bool, len(sample.Absent))
		for j, a := range sample.Absent {
			afield := fmt.Sprintf("%s.absent[%d]", field, j)
			switch {
			case strings.TrimSpace(a.Field) == "":
				return &ValidationError{Field: afield + ".field", Message: "field is required"}
			case !validAbsentReasons[a.Reason]:
				return &ValidationError{Field: afield + ".reason", Message: fmt.Sprintf("unknown reason %q", a.Reason)}
			case absent[a.Field]:
				return &ValidationError{Field: afield, Message: fmt.Sprintf("field %q appears twice", a.Field)}
			}
			absent[a.Field] = true
		}
		key := sample.Flow + "\x00" + sample.Variant
		if seen[key] {
			return &ValidationError{Field: field, Message: fmt.Sprintf("flow %q variant %q appears twice", sample.Flow, sample.Variant)}
		}
		seen[key] = true
	}
	return nil
}
