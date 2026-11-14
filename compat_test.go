package aurora_test

// Backward-compatibility tests. They use only the exported API, the way the
// admin service does, and pin what v1.0.15 shipped: field names, JSON tags
// and field types. Adding fields is fine; removing, renaming or retyping one
// breaks callers that build these structs or stored their JSON (the admin
// service keeps CallbackTransactionRequest, ProcessRequest and
// ProcessResponse in its task table).

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	aurora "github.com/kintsdev/aurora-go-sdk"
)

type frozenField struct {
	name string // Go field name
	tag  string // full json tag
	typ  string // reflect type string
}

// v1Fields is generated from models.go at v1.0.15. Do not edit entries;
// append only when a new field ships in a release.
var v1Fields = map[string][]frozenField{
	"ProcessRequest": {
		{"RuleID", "rule_id,omitempty", "string"},
		{"RulesetID", "ruleset_id,omitempty", "string"},
		{"TransactionID", "transaction_id,omitempty", "string"},
		{"LifecycleRootTransactionID", "lifecycle_root_transaction_id,omitempty", "string"},
		{"Transaction", "transaction,omitempty", "*aurora.Transaction"},
	},
	"Transaction": {
		{"AccountChange", "account_change,omitempty", "*aurora.AccountChange"},
		{"AccountLogin", "account_login,omitempty", "*aurora.AccountLogin"},
		{"AccountOpening", "account_opening,omitempty", "*aurora.AccountOpening"},
		{"BNPL", "bnpl,omitempty", "*aurora.BNPL"},
		{"Card", "card,omitempty", "*aurora.Card"},
		{"Common", "common,omitempty", "*aurora.Common"},
		{"Crypto", "crypto,omitempty", "*aurora.Crypto"},
		{"Deposit", "deposit,omitempty", "*aurora.Deposit"},
		{"Invoice", "invoice,omitempty", "*aurora.Invoice"},
		{"Loan", "loan,omitempty", "*aurora.Loan"},
		{"P2P", "p2p,omitempty", "*aurora.P2P"},
		{"Refund", "refund,omitempty", "*aurora.Refund"},
		{"Transfer", "transfer,omitempty", "*aurora.Transfer"},
		{"Withdrawal", "withdrawal,omitempty", "*aurora.Withdrawal"},
		{"AccountAgeDays", "account_age_days,omitempty", "string"},
		{"CustomerAccountType", "customer_account_type,omitempty", "string"},
		{"CustomerAgeYears", "customer_age_years,omitempty", "string"},
		{"DeclinedCount", "declined_count,omitempty", "string"},
		{"FirstTransactionAgeDays", "first_transaction_age_days,omitempty", "string"},
		{"IncomeMultiplier", "income_multiplier,omitempty", "string"},
		{"IPCity", "ip_city,omitempty", "string"},
		{"IPCountry", "ip_country,omitempty", "string"},
		{"IsFirstTransfer", "is_first_transfer,omitempty", "string"},
		{"IsNewDevice", "is_new_device,omitempty", "string"},
		{"IsNewIP", "is_new_ip,omitempty", "string"},
		{"IsUnusualLocation", "is_unusual_location,omitempty", "string"},
		{"PasswordChangedRecently", "password_changed_recently,omitempty", "string"},
		{"ProfileCompletion", "profile_completion,omitempty", "string"},
		{"RefundCount", "refund_count,omitempty", "string"},
		{"RefundRatio", "refund_ratio,omitempty", "string"},
		{"RegisteredIncome", "registered_income,omitempty", "string"},
		{"TotalAmount24h", "total_amount_24h,omitempty", "string"},
		{"TotalAmount7d", "total_amount_7d,omitempty", "string"},
		{"TransactionHour", "transaction_hour,omitempty", "string"},
		{"TransactionType", "transaction_type,omitempty", "string"},
		{"TransferCount24h", "transfer_count_24h,omitempty", "string"},
		{"Type", "type", "string"},
		{"UniqueRecipients", "unique_recipients,omitempty", "string"},
	},
	"Common": {
		{"Amount", "amount,omitempty", "float64"},
		{"Balance", "balance,omitempty", "float64"},
		{"BrowserAgent", "browser_agent,omitempty", "string"},
		{"Category", "category,omitempty", "string"},
		{"ConnectionType", "connection_type,omitempty", "string"},
		{"Country", "country,omitempty", "string"},
		{"Currency", "currency,omitempty", "string"},
		{"Date", "date,omitempty", "string"},
		{"Description", "description,omitempty", "string"},
		{"DeviceFingerprint", "device_fingerprint,omitempty", "string"},
		{"DeviceID", "device_id,omitempty", "string"},
		{"FirstName", "first_name,omitempty", "string"},
		{"LastName", "last_name,omitempty", "string"},
		{"BillingName", "billing_name,omitempty", "string"},
		{"IdentityNumber", "identity_number,omitempty", "string"},
		{"Email", "email,omitempty", "string"},
		{"IPAddress", "ip_address,omitempty", "string"},
		{"LastLoginIP", "last_login_ip,omitempty", "string"},
		{"LastLoginTime", "last_login_time,omitempty", "string"},
		{"Latitude", "latitude,omitempty", "string"},
		{"LocationCity", "location_city,omitempty", "string"},
		{"LocationCountry", "location_country,omitempty", "string"},
		{"LocationRegion", "location_region,omitempty", "string"},
		{"LocationRegionCode", "location_region_code,omitempty", "string"},
		{"LocationSource", "location_source,omitempty", "string"},
		{"Longitude", "longitude,omitempty", "string"},
		{"MerchantID", "merchant_id,omitempty", "string"},
		{"MerchantName", "merchant_name,omitempty", "string"},
		{"PaymentID", "payment_id,omitempty", "string"},
		{"Phone", "phone,omitempty", "string"},
		{"ReferenceID", "reference_id,omitempty", "string"},
		{"UserID", "user_id,omitempty", "string"},
		{"ShippingAddress", "shipping_address,omitempty", "*aurora.ShippingAddress"},
	},
	"ShippingAddress": {
		{"Address", "address", "string"},
		{"City", "city", "string"},
		{"Country", "country", "string"},
		{"ZipCode", "zip_code,omitempty", "string"},
	},
	"Card": {
		{"BinNumber", "bin_number,omitempty", "string"},
		{"Holder", "holder,omitempty", "string"},
		{"Issuer", "issuer,omitempty", "string"},
		{"LastFour", "last_four,omitempty", "string"},
		{"MCC", "mcc,omitempty", "string"},
		{"Network", "network,omitempty", "string"},
		{"Token", "token,omitempty", "string"},
		{"Type", "type,omitempty", "string"},
	},
	"Transfer": {
		{"ExchangeRate", "exchange_rate,omitempty", "string"},
		{"PaymentMethod", "payment_method,omitempty", "string"},
		{"ReceiverAddress", "receiver_address,omitempty", "string"},
		{"ReceiverCountry", "receiver_country,omitempty", "string"},
		{"ReceiverIBAN", "receiver_iban,omitempty", "string"},
		{"ReceiverIdentityPassport", "receiver_identity_passport,omitempty", "string"},
		{"ReceiverName", "receiver_name,omitempty", "string"},
		{"ReceiverSurname", "receiver_surname,omitempty", "string"},
		{"ReceiverWalletID", "receiver_wallet_id,omitempty", "string"},
		{"ReceiverIdentityNumber", "receiver_identity_number,omitempty", "string"},
		{"ReceiverPhone", "receiver_phone,omitempty", "string"},
		{"ReceiverEmail", "receiver_email,omitempty", "string"},
		{"Relationship", "relationship,omitempty", "string"},
		{"SenderAddress", "sender_address,omitempty", "string"},
		{"SenderCountry", "sender_country,omitempty", "string"},
		{"SenderIBAN", "sender_iban,omitempty", "string"},
		{"SenderIdentityPassport", "sender_identity_passport,omitempty", "string"},
		{"SenderName", "sender_name,omitempty", "string"},
		{"SenderSurname", "sender_surname,omitempty", "string"},
		{"SenderWalletID", "sender_wallet_id,omitempty", "string"},
		{"SenderIdentityNumber", "sender_identity_number,omitempty", "string"},
		{"SenderPhone", "sender_phone,omitempty", "string"},
		{"SenderEmail", "sender_email,omitempty", "string"},
		{"SourceOfFunds", "source_of_funds,omitempty", "string"},
		{"TargetCurrency", "target_currency,omitempty", "string"},
		{"TransferPurpose", "transfer_purpose,omitempty", "string"},
		{"TransferType", "transfer_type,omitempty", "string"},
	},
	"AccountChange": {
		{"ChangeType", "change_type,omitempty", "string"},
		{"PreviousValueHash", "previous_value_hash,omitempty", "string"},
		{"TimeSinceLastChange", "time_since_last_change,omitempty", "string"},
		{"VerificationMethod", "verification_method,omitempty", "string"},
	},
	"AccountLogin": {
		{"FailedAttempts", "failed_attempts,omitempty", "string"},
		{"LoginMethod", "login_method,omitempty", "string"},
		{"LoginStatus", "login_status,omitempty", "string"},
		{"MFAMethod", "mfa_method,omitempty", "string"},
		{"MFAUsed", "mfa_used,omitempty", "string"},
		{"SessionID", "session_id,omitempty", "string"},
	},
	"AccountOpening": {
		{"DocumentType", "document_type,omitempty", "string"},
		{"IdentityVerificationStatus", "identity_verification_status,omitempty", "string"},
		{"RegistrationMethod", "registration_method,omitempty", "string"},
	},
	"BNPL": {
		{"BNPLProvider", "bnpl_provider,omitempty", "string"},
		{"InstallmentCount", "installment_count,omitempty", "string"},
		{"OutstandingBNPLAmount", "outstanding_bnpl_amount,omitempty", "string"},
	},
	"Crypto": {
		{"CryptoAmount", "crypto_amount,omitempty", "string"},
		{"CryptoCurrency", "crypto_currency,omitempty", "string"},
		{"ExchangeName", "exchange_name,omitempty", "string"},
		{"IsSmartContract", "is_smart_contract,omitempty", "string"},
		{"WalletAddress", "wallet_address,omitempty", "string"},
	},
	"Deposit": {
		{"CheckNumber", "check_number,omitempty", "string"},
		{"DepositMethod", "deposit_method,omitempty", "string"},
		{"DepositSource", "deposit_source,omitempty", "string"},
		{"IsRemoteDeposit", "is_remote_deposit,omitempty", "string"},
	},
	"Invoice": {
		{"InvoiceDueDate", "invoice_due_date,omitempty", "string"},
		{"InvoiceNumber", "invoice_number,omitempty", "string"},
		{"IsRecurringVendor", "is_recurring_vendor,omitempty", "string"},
		{"PurchaseOrderID", "purchase_order_id,omitempty", "string"},
		{"VendorID", "vendor_id,omitempty", "string"},
		{"VendorName", "vendor_name,omitempty", "string"},
	},
	"Loan": {
		{"CreditScore", "credit_score,omitempty", "string"},
		{"DebtToIncomeRatio", "debt_to_income_ratio,omitempty", "string"},
		{"EmploymentStatus", "employment_status,omitempty", "string"},
		{"LoanAmount", "loan_amount,omitempty", "string"},
		{"LoanTerm", "loan_term,omitempty", "string"},
		{"LoanType", "loan_type,omitempty", "string"},
	},
	"P2P": {
		{"PaymentNote", "payment_note,omitempty", "string"},
		{"Platform", "platform,omitempty", "string"},
		{"RecipientAccountAge", "recipient_account_age,omitempty", "string"},
	},
	"Refund": {
		{"DaysSincePurchase", "days_since_purchase,omitempty", "string"},
		{"OriginalTransactionID", "original_transaction_id,omitempty", "string"},
		{"RefundMethod", "refund_method,omitempty", "string"},
		{"RefundReason", "refund_reason,omitempty", "string"},
	},
	"Withdrawal": {
		{"AccountType", "account_type,omitempty", "string"},
		{"ATMID", "atm_id,omitempty", "string"},
		{"DailyWithdrawalCount", "daily_withdrawal_count,omitempty", "string"},
		{"WithdrawalMethod", "withdrawal_method,omitempty", "string"},
	},
	"RuleWarning": {
		{"Code", "code", "string"},
		{"Message", "message", "string"},
	},
	"ProcessResponse": {
		{"Warnings", "warnings,omitempty", "[]aurora.RuleWarning"},
		{"TransactionID", "transaction_id,omitempty", "string"},
		{"Allow", "allow", "bool"},
		{"AllowMessage", "allow_message,omitempty", "string"},
		{"Error", "error", "bool"},
		{"ErrorMessage", "error_message,omitempty", "string"},
		{"ExecutionTime", "execution_time", "time.Duration"},
		{"InspectMessage", "inspect_message,omitempty", "string"},
		{"NeedInspect", "need_inspect", "bool"},
		{"RejectMessage", "reject_message,omitempty", "string"},
		{"Rejected", "rejected", "bool"},
		{"Score", "score", "int"},
	},
	"CallbackTransactionRequest": {
		{"Message", "message", "string"},
		{"PaymentID", "payment_id", "string"},
		{"Status", "status", "string"},
	},
	"CallbackTransactionResponse": {
		{"Message", "message", "string"},
	},
	"ErrorResponse": {
		{"Error", "error", "string"},
	},
}

var exportedTypes = map[string]reflect.Type{
	"ProcessRequest":              reflect.TypeOf(aurora.ProcessRequest{}),
	"Transaction":                 reflect.TypeOf(aurora.Transaction{}),
	"Common":                      reflect.TypeOf(aurora.Common{}),
	"ShippingAddress":             reflect.TypeOf(aurora.ShippingAddress{}),
	"Card":                        reflect.TypeOf(aurora.Card{}),
	"Transfer":                    reflect.TypeOf(aurora.Transfer{}),
	"AccountChange":               reflect.TypeOf(aurora.AccountChange{}),
	"AccountLogin":                reflect.TypeOf(aurora.AccountLogin{}),
	"AccountOpening":              reflect.TypeOf(aurora.AccountOpening{}),
	"BNPL":                        reflect.TypeOf(aurora.BNPL{}),
	"Crypto":                      reflect.TypeOf(aurora.Crypto{}),
	"Deposit":                     reflect.TypeOf(aurora.Deposit{}),
	"Invoice":                     reflect.TypeOf(aurora.Invoice{}),
	"Loan":                        reflect.TypeOf(aurora.Loan{}),
	"P2P":                         reflect.TypeOf(aurora.P2P{}),
	"Refund":                      reflect.TypeOf(aurora.Refund{}),
	"Withdrawal":                  reflect.TypeOf(aurora.Withdrawal{}),
	"RuleWarning":                 reflect.TypeOf(aurora.RuleWarning{}),
	"ProcessResponse":             reflect.TypeOf(aurora.ProcessResponse{}),
	"CallbackTransactionRequest":  reflect.TypeOf(aurora.CallbackTransactionRequest{}),
	"CallbackTransactionResponse": reflect.TypeOf(aurora.CallbackTransactionResponse{}),
	"ErrorResponse":               reflect.TypeOf(aurora.ErrorResponse{}),
}

func TestV1FieldsUnchanged(t *testing.T) {
	for typeName, fields := range v1Fields {
		rt, ok := exportedTypes[typeName]
		if !ok {
			t.Errorf("type %s is no longer checked; add it to exportedTypes", typeName)
			continue
		}
		for _, want := range fields {
			got, ok := rt.FieldByName(want.name)
			if !ok {
				t.Errorf("%s.%s was removed or renamed", typeName, want.name)
				continue
			}
			if tag := got.Tag.Get("json"); tag != want.tag {
				t.Errorf("%s.%s json tag = %q, want %q", typeName, want.name, tag, want.tag)
			}
			if typ := got.Type.String(); typ != want.typ {
				t.Errorf("%s.%s type = %s, want %s", typeName, want.name, typ, want.typ)
			}
		}
	}
}

func TestV1ConstantsUnchanged(t *testing.T) {
	constants := map[string]string{
		aurora.TransactionTypeCard:                     "card",
		aurora.TransactionTypeTransfer:                 "transfer",
		aurora.TransactionTypeRemittance:               "remittance",
		aurora.TransactionTypeAccountOpening:           "account_opening",
		aurora.TransactionTypeAccountLogin:             "account_login",
		aurora.TransactionTypeAccountChange:            "account_change",
		aurora.TransactionTypeRefund:                   "refund",
		aurora.TransactionTypeWithdrawal:               "withdrawal",
		aurora.TransactionTypeDeposit:                  "deposit",
		aurora.TransactionTypeP2P:                      "p2p",
		aurora.TransactionTypeCrypto:                   "crypto",
		aurora.TransactionTypeBNPL:                     "bnpl",
		aurora.TransactionTypeLoan:                     "loan",
		aurora.TransactionTypeInvoice:                  "invoice",
		aurora.TransferTypeWalletToWallet:              "wallet_to_wallet",
		aurora.TransferTypeWalletToIBAN:                "wallet_to_iban",
		aurora.TransferTypeIBANToWallet:                "iban_to_wallet",
		aurora.WithdrawalMethodMoneyTransferCollection: "money_transfer_collection",
		aurora.CustomerAccountTypeIndividual:           "individual",
		aurora.CustomerAccountTypeOrganization:         "organization",
	}
	for got, want := range constants {
		if got != want {
			t.Errorf("constant value %q, want %q", got, want)
		}
	}
	if aurora.TransferTypeRemittance != "remittance" {
		t.Errorf("TransferTypeRemittance = %q", aurora.TransferTypeRemittance)
	}
}

// Payloads in the shape the admin service stored with v1.0.15 must still
// decode into the current types.
func TestDecodesStoredV1Payloads(t *testing.T) {
	t.Run("payment outcome task", func(t *testing.T) {
		stored := `{"provider_id":"0b0c6a9e-8f1d-4d6e-9a43-0c8f7d2b1e55","provider_organization_id":"9c2f5a41-6b0e-4c8a-9f31-2d7b8e4a1c05","request":{"message":"failed","payment_id":"5d1c2b8e-0f9a-4c3e-8b7d-6a5e4f3d2c1b","status":"failed"}}`
		var task struct {
			Request aurora.CallbackTransactionRequest `json:"request"`
		}
		if err := json.Unmarshal([]byte(stored), &task); err != nil {
			t.Fatal(err)
		}
		want := aurora.CallbackTransactionRequest{Message: "failed", PaymentID: "5d1c2b8e-0f9a-4c3e-8b7d-6a5e4f3d2c1b", Status: aurora.CallbackStatusFailed}
		if task.Request != want {
			t.Errorf("request = %+v, want %+v", task.Request, want)
		}
	})

	t.Run("bank transfer review task", func(t *testing.T) {
		stored := `{
			"request": {"ruleset_id":"bank-warning","transaction":{"type":"transfer","customer_account_type":"individual","account_age_days":"30",
				"common":{"amount":1500.5,"currency":"TRY","payment_id":"pay-1","billing_name":"Jane Doe","shipping_address":{"address":"Main St 1","city":"Istanbul","country":"TR"}},
				"transfer":{"sender_name":"Jane Doe","sender_iban":"TR000000000000000000000001"}}},
			"result": {"transaction_id":"tx-1","allow":false,"error":false,"execution_time":1500000,"inspect_message":"name mismatch","need_inspect":true,"rejected":false,"score":40,
				"warnings":[{"code":"card_holder_name_mismatch","message":"name differs"}]}
		}`
		var task struct {
			Request *aurora.ProcessRequest  `json:"request"`
			Result  *aurora.ProcessResponse `json:"result,omitempty"`
		}
		if err := json.Unmarshal([]byte(stored), &task); err != nil {
			t.Fatal(err)
		}
		tx := task.Request.Transaction
		if task.Request.RulesetID != "bank-warning" || tx.Type != aurora.TransactionTypeTransfer || tx.Common.Amount != 1500.5 ||
			tx.Common.ShippingAddress.City != "Istanbul" || tx.Transfer.SenderIBAN != "TR000000000000000000000001" {
			t.Errorf("request decoded wrong: %+v", task.Request)
		}
		res := task.Result
		if !res.NeedInspect || res.Allow || res.Challenge || res.Score != 40 ||
			res.ExecutionTime != 1500*time.Microsecond || len(res.Warnings) != 1 {
			t.Errorf("result decoded wrong: %+v", res)
		}
	})
}

// Every key v1.0.15 always wrote for ProcessResponse is still written, so
// stored JSON and logs keep their shape; new keys may appear.
func TestProcessResponseKeepsV1Keys(t *testing.T) {
	data, err := json.Marshal(aurora.ProcessResponse{})
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal(data, &keys); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"allow", "error", "execution_time", "need_inspect", "rejected", "score"} {
		if _, ok := keys[k]; !ok {
			t.Errorf("key %q missing from %s", k, data)
		}
	}
}

// Mirrors how the admin service calls the SDK. If this stops compiling, an
// admin build would stop compiling too.
func TestAdminUsagePatterns(t *testing.T) {
	var gotPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		_, _ = io.Copy(io.Discard, r.Body)
		switch r.URL.Path {
		case "/api/v1/process":
			_, _ = io.WriteString(w, `{"transaction_id":"tx-1","need_inspect":true,"inspect_message":"review"}`)
		default:
			_, _ = io.WriteString(w, `{"message":"Callback received"}`)
		}
	}))
	defer srv.Close()

	provider := struct{ Token, Host, TriggeredRuleSetKey string }{"token", srv.URL, "ruleset-1"}
	client := aurora.NewClient(provider.Token, aurora.WithBaseURL(provider.Host))
	ctx := context.Background()

	requests := []aurora.ProcessRequest{
		{
			RulesetID: provider.TriggeredRuleSetKey,
			Transaction: &aurora.Transaction{
				Type:                aurora.TransactionTypeCard,
				CustomerAccountType: aurora.CustomerAccountTypeIndividual,
				Common: &aurora.Common{
					Amount: 99.9, Currency: "TRY", PaymentID: "pay-1", MerchantID: "m-1", BillingName: "Jane Doe",
					ShippingAddress: &aurora.ShippingAddress{Address: "Main St 1", City: "Istanbul", Country: "TR"},
				},
				Card: &aurora.Card{BinNumber: "454360", Holder: "Jane Doe", LastFour: "0001", Network: "visa", Token: "tok"},
			},
		},
		{
			RulesetID:                  provider.TriggeredRuleSetKey,
			LifecycleRootTransactionID: "9c2f5a41-6b0e-4c8a-9f31-2d7b8e4a1c05",
			Transaction: &aurora.Transaction{
				Type:                aurora.TransactionTypeWithdrawal,
				CustomerAccountType: aurora.CustomerAccountTypeOrganization,
				Common:              &aurora.Common{Amount: 10},
				Withdrawal:          &aurora.Withdrawal{WithdrawalMethod: aurora.WithdrawalMethodMoneyTransferCollection},
			},
		},
		{
			RulesetID: provider.TriggeredRuleSetKey,
			Transaction: &aurora.Transaction{
				Type:     aurora.TransactionTypeTransfer,
				Common:   &aurora.Common{Amount: 10},
				Transfer: &aurora.Transfer{TransferType: aurora.TransferTypeRemittance, SenderName: "Jane", ReceiverName: "John"},
			},
		},
		{
			RulesetID: provider.TriggeredRuleSetKey,
			Transaction: &aurora.Transaction{
				Type:    aurora.TransactionTypeDeposit,
				Common:  &aurora.Common{Amount: 10},
				Deposit: &aurora.Deposit{DepositMethod: "bank_transfer"},
			},
		},
	}
	for i := range requests {
		resp, err := client.Process.Execute(ctx, &requests[i])
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		switch {
		case resp.Rejected, resp.Error:
			t.Errorf("request %d: unexpected decision %+v", i, resp)
		case resp.NeedInspect:
			if resp.InspectMessage != "review" || resp.TransactionID != "tx-1" {
				t.Errorf("request %d: %+v", i, resp)
			}
		}
	}

	// Admin builds a response literal when Aurora is skipped.
	skipped := &aurora.ProcessResponse{Allow: true}
	if !skipped.Allow {
		t.Error("literal lost Allow")
	}

	if _, err := client.Callback.Transaction(ctx, &aurora.CallbackTransactionRequest{
		PaymentID: "pay-1",
		Status:    "failed",
		Message:   "Order has failed",
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}

	var apiErr *aurora.APIError
	if errors.As(errors.New("plain"), &apiErr) {
		t.Error("errors.As matched a plain error")
	}
	if want := 5; len(gotPaths) != want || !strings.HasSuffix(gotPaths[want-1], "/callback/transaction") {
		t.Errorf("paths = %v", gotPaths)
	}
}
