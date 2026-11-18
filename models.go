package aurora

import "time"

// ProcessRequest is the input to ProcessService.Execute.
type ProcessRequest struct {
	// RuleID selects a single rule. It takes precedence over RulesetID.
	RuleID string `json:"rule_id,omitempty"`
	// RulesetID selects a ruleset. Used only when RuleID is empty.
	RulesetID string `json:"ruleset_id,omitempty"`
	// TransactionID re-evaluates a transaction Aurora already stored. Leave it
	// empty to submit a new transaction in Transaction.
	TransactionID string `json:"transaction_id,omitempty"`
	// LifecycleRootTransactionID links a remittance collection to the remittance
	// send that opened its lifecycle. Set it only on a remittance collection
	// request. On re-evaluation it must match the link already stored.
	LifecycleRootTransactionID string `json:"lifecycle_root_transaction_id,omitempty"`
	// Transaction is the data to evaluate. Required when TransactionID is
	// empty; ignored by Aurora when TransactionID is set.
	Transaction *Transaction `json:"transaction,omitempty"`
}

// Transaction contains all the data fields that can be evaluated by rules.
type Transaction struct {
	// Nested sub-objects
	AccountChange  *AccountChange  `json:"account_change,omitempty"`
	AccountLogin   *AccountLogin   `json:"account_login,omitempty"`
	AccountOpening *AccountOpening `json:"account_opening,omitempty"`
	BNPL           *BNPL           `json:"bnpl,omitempty"`
	Booking        *Booking        `json:"booking,omitempty"`
	Card           *Card           `json:"card,omitempty"`
	Common         *Common         `json:"common,omitempty"`
	Crypto         *Crypto         `json:"crypto,omitempty"`
	Deposit        *Deposit        `json:"deposit,omitempty"`
	Invoice        *Invoice        `json:"invoice,omitempty"`
	Loan           *Loan           `json:"loan,omitempty"`
	P2P            *P2P            `json:"p2p,omitempty"`
	Refund         *Refund         `json:"refund,omitempty"`
	Transfer       *Transfer       `json:"transfer,omitempty"`
	Withdrawal     *Withdrawal     `json:"withdrawal,omitempty"`

	// Attributes are caller-defined key/value pairs that rules read as attr.<key>.
	// Keys match ^[a-z][a-z0-9_]{0,63}$; at most MaxAttributes keys and MaxAttributeValueLength characters per value.
	Attributes map[string]string `json:"attributes,omitempty"`

	// Top-level scalar fields
	AccountAgeDays          string `json:"account_age_days,omitempty"`
	CustomerAccountType     string `json:"customer_account_type,omitempty"`
	CustomerAgeYears        string `json:"customer_age_years,omitempty"`
	DeclinedCount           string `json:"declined_count,omitempty"`
	FirstTransactionAgeDays string `json:"first_transaction_age_days,omitempty"`
	IncomeMultiplier        string `json:"income_multiplier,omitempty"`
	IPCity                  string `json:"ip_city,omitempty"`
	IPCountry               string `json:"ip_country,omitempty"`
	IsFirstTransfer         string `json:"is_first_transfer,omitempty"`
	IsNewDevice             string `json:"is_new_device,omitempty"`
	IsNewIP                 string `json:"is_new_ip,omitempty"`
	IsUnusualLocation       string `json:"is_unusual_location,omitempty"`
	PasswordChangedRecently string `json:"password_changed_recently,omitempty"`
	ProfileCompletion       string `json:"profile_completion,omitempty"`
	RefundCount             string `json:"refund_count,omitempty"`
	RefundRatio             string `json:"refund_ratio,omitempty"`
	RegisteredIncome        string `json:"registered_income,omitempty"`
	TotalAmount24h          string `json:"total_amount_24h,omitempty"`
	TotalAmount7d           string `json:"total_amount_7d,omitempty"`
	TransactionHour         string `json:"transaction_hour,omitempty"`
	TransactionType         string `json:"transaction_type,omitempty"`
	TransferCount24h        string `json:"transfer_count_24h,omitempty"`
	Type                    string `json:"type"`
	UniqueRecipients        string `json:"unique_recipients,omitempty"`
}

// Common holds core transaction and user identification fields.
type Common struct {
	Amount             float64 `json:"amount,omitempty"`
	Balance            float64 `json:"balance,omitempty"`
	BrowserAgent       string  `json:"browser_agent,omitempty"`
	Category           string  `json:"category,omitempty"`
	ConnectionType     string  `json:"connection_type,omitempty"`
	Country            string  `json:"country,omitempty"`
	Currency           string  `json:"currency,omitempty"`
	Date               string  `json:"date,omitempty"`
	Description        string  `json:"description,omitempty"`
	DeviceFingerprint  string  `json:"device_fingerprint,omitempty"`
	DeviceID           string  `json:"device_id,omitempty"`
	FirstName          string  `json:"first_name,omitempty" example:"John"`
	LastName           string  `json:"last_name,omitempty" example:"Doe"`
	BillingName        string  `json:"billing_name,omitempty"`
	IdentityNumber     string  `json:"identity_number,omitempty" example:"123456789"`
	Email              string  `json:"email,omitempty"`
	IPAddress          string  `json:"ip_address,omitempty"`
	LastLoginIP        string  `json:"last_login_ip,omitempty"`
	LastLoginTime      string  `json:"last_login_time,omitempty"`
	Latitude           string  `json:"latitude,omitempty"`
	LocationCity       string  `json:"location_city,omitempty"`
	LocationCountry    string  `json:"location_country,omitempty"`
	LocationRegion     string  `json:"location_region,omitempty"`
	LocationRegionCode string  `json:"location_region_code,omitempty"`
	LocationSource     string  `json:"location_source,omitempty"`
	Longitude          string  `json:"longitude,omitempty"`
	MerchantID         string  `json:"merchant_id,omitempty"`
	MerchantName       string  `json:"merchant_name,omitempty"`
	PaymentID          string  `json:"payment_id,omitempty"`
	Phone              string  `json:"phone,omitempty"`
	ReferenceID        string  `json:"reference_id,omitempty"`
	UserID             string  `json:"user_id,omitempty"`

	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty"`
}

// ShippingAddress identifies the delivery location used by an order.
type ShippingAddress struct {
	Address string `json:"address"`
	City    string `json:"city"`
	Country string `json:"country"`
	ZipCode string `json:"zip_code,omitempty"`
}

// Card holds payment card details.
type Card struct {
	BinNumber string `json:"bin_number,omitempty"`
	Holder    string `json:"holder,omitempty"`
	Issuer    string `json:"issuer,omitempty"`
	LastFour  string `json:"last_four,omitempty"`
	MCC       string `json:"mcc,omitempty"`
	Network   string `json:"network,omitempty"`
	Token     string `json:"token,omitempty"`
	Type      string `json:"type,omitempty"`
}

// Transfer holds international money transfer fields.
type Transfer struct {
	ExchangeRate             string `json:"exchange_rate,omitempty"`
	PaymentMethod            string `json:"payment_method,omitempty"`
	ReceiverAddress          string `json:"receiver_address,omitempty"`
	ReceiverCountry          string `json:"receiver_country,omitempty"`
	ReceiverIBAN             string `json:"receiver_iban,omitempty"`
	ReceiverIdentityPassport string `json:"receiver_identity_passport,omitempty"`
	ReceiverName             string `json:"receiver_name,omitempty"`
	ReceiverSurname          string `json:"receiver_surname,omitempty"`
	ReceiverWalletID         string `json:"receiver_wallet_id,omitempty"`
	ReceiverIdentityNumber   string `json:"receiver_identity_number,omitempty" example:"987654321"`
	ReceiverPhone            string `json:"receiver_phone,omitempty" example:"+0987654321"`
	ReceiverEmail            string `json:"receiver_email,omitempty" example:"john@doe.com"`
	Relationship             string `json:"relationship,omitempty"`
	SenderAddress            string `json:"sender_address,omitempty"`
	SenderCountry            string `json:"sender_country,omitempty"`
	SenderIBAN               string `json:"sender_iban,omitempty"`
	SenderIdentityPassport   string `json:"sender_identity_passport,omitempty"`
	SenderName               string `json:"sender_name,omitempty"`
	SenderSurname            string `json:"sender_surname,omitempty"`
	SenderWalletID           string `json:"sender_wallet_id,omitempty"`
	SenderIdentityNumber     string `json:"sender_identity_number,omitempty" example:"123456789"`
	SenderPhone              string `json:"sender_phone,omitempty" example:"+1234567890"`
	SenderEmail              string `json:"sender_email,omitempty" example:"john@doe.com"`
	SourceOfFunds            string `json:"source_of_funds,omitempty"`
	TargetCurrency           string `json:"target_currency,omitempty"`
	TransferPurpose          string `json:"transfer_purpose,omitempty"`
	TransferType             string `json:"transfer_type,omitempty"`
}

// AccountChange holds account modification event data.
type AccountChange struct {
	ChangeType          string `json:"change_type,omitempty"`
	PreviousValueHash   string `json:"previous_value_hash,omitempty"`
	TimeSinceLastChange string `json:"time_since_last_change,omitempty"`
	VerificationMethod  string `json:"verification_method,omitempty"`
}

// AccountLogin holds login event data.
type AccountLogin struct {
	FailedAttempts string `json:"failed_attempts,omitempty"`
	LoginMethod    string `json:"login_method,omitempty"`
	LoginStatus    string `json:"login_status,omitempty"`
	MFAMethod      string `json:"mfa_method,omitempty"`
	MFAUsed        string `json:"mfa_used,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
}

// AccountOpening holds account registration data.
type AccountOpening struct {
	DocumentType               string `json:"document_type,omitempty"`
	IdentityVerificationStatus string `json:"identity_verification_status,omitempty"`
	RegistrationMethod         string `json:"registration_method,omitempty"`
}

// BNPL holds buy-now-pay-later transaction data.
type BNPL struct {
	BNPLProvider          string `json:"bnpl_provider,omitempty"`
	InstallmentCount      string `json:"installment_count,omitempty"`
	OutstandingBNPLAmount string `json:"outstanding_bnpl_amount,omitempty"`
}

// Crypto holds cryptocurrency transaction data.
type Crypto struct {
	CryptoAmount    string `json:"crypto_amount,omitempty"`
	CryptoCurrency  string `json:"crypto_currency,omitempty"`
	ExchangeName    string `json:"exchange_name,omitempty"`
	IsSmartContract string `json:"is_smart_contract,omitempty"`
	WalletAddress   string `json:"wallet_address,omitempty"`
}

// Deposit holds deposit transaction data.
type Deposit struct {
	CheckNumber     string `json:"check_number,omitempty"`
	DepositMethod   string `json:"deposit_method,omitempty"`
	DepositSource   string `json:"deposit_source,omitempty"`
	IsRemoteDeposit string `json:"is_remote_deposit,omitempty"`
}

// Invoice holds invoice and vendor payment data.
type Invoice struct {
	InvoiceDueDate    string `json:"invoice_due_date,omitempty"`
	InvoiceNumber     string `json:"invoice_number,omitempty"`
	IsRecurringVendor string `json:"is_recurring_vendor,omitempty"`
	PurchaseOrderID   string `json:"purchase_order_id,omitempty"`
	VendorID          string `json:"vendor_id,omitempty"`
	VendorName        string `json:"vendor_name,omitempty"`
}

// Booking holds reservation data; Aurora derives lead time, nights and guest name match from it.
type Booking struct {
	BookingReference string `json:"booking_reference,omitempty"`
	BookingType      string `json:"booking_type,omitempty"`
	BookingChannel   string `json:"booking_channel,omitempty"`
	PropertyCountry  string `json:"property_country,omitempty"`
	CheckInDate      string `json:"check_in_date,omitempty"`
	CheckOutDate     string `json:"check_out_date,omitempty"`
	GuestName        string `json:"guest_name,omitempty"`
	GuestCount       string `json:"guest_count,omitempty"`
	RoomCount        string `json:"room_count,omitempty"`
	IsRefundable     string `json:"is_refundable,omitempty"`
	BookingTotal     string `json:"booking_total,omitempty"`
}

// Loan holds loan application data.
type Loan struct {
	CreditScore       string `json:"credit_score,omitempty"`
	DebtToIncomeRatio string `json:"debt_to_income_ratio,omitempty"`
	EmploymentStatus  string `json:"employment_status,omitempty"`
	LoanAmount        string `json:"loan_amount,omitempty"`
	LoanTerm          string `json:"loan_term,omitempty"`
	LoanType          string `json:"loan_type,omitempty"`
}

// P2P holds peer-to-peer payment data.
type P2P struct {
	PaymentNote         string `json:"payment_note,omitempty"`
	Platform            string `json:"platform,omitempty"`
	RecipientAccountAge string `json:"recipient_account_age,omitempty"`
}

// Refund holds refund transaction data.
type Refund struct {
	DaysSincePurchase     string `json:"days_since_purchase,omitempty"`
	OriginalTransactionID string `json:"original_transaction_id,omitempty"`
	RefundMethod          string `json:"refund_method,omitempty"`
	RefundReason          string `json:"refund_reason,omitempty"`
}

// Withdrawal holds withdrawal transaction data.
type Withdrawal struct {
	AccountType          string `json:"account_type,omitempty"`
	ATMID                string `json:"atm_id,omitempty"`
	DailyWithdrawalCount string `json:"daily_withdrawal_count,omitempty"`
	WithdrawalMethod     string `json:"withdrawal_method,omitempty"`
}

// RuleWarning is a risk finding that does not block, hold or challenge a
// payment on its own.
type RuleWarning struct {
	// Code identifies the warning. It defaults to the rule node's ID when the
	// rule does not set one.
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ProcessResponse is the decision returned by ProcessService.Execute.
//
// The decision is carried by four flags: Rejected, Challenge, NeedInspect and
// Allow. For a ruleset, Aurora sets exactly one of them, using that order of
// precedence. For a single rule, more than one can be true, because each
// output node of the rule sets its own flag. Check the flags in the same
// order: Rejected, Challenge, NeedInspect, Allow.
//
// Error means at least one rule failed to evaluate. Do not treat the payment
// as allowed when Error is true. For a ruleset, Aurora then clears Allow and
// keeps any Rejected, Challenge or NeedInspect result. If no rule produced
// one, all four flags are false.
type ProcessResponse struct {
	// Warnings lists risk findings that do not block, hold or challenge the
	// payment.
	Warnings []RuleWarning `json:"warnings,omitempty"`
	// TransactionID is Aurora's ID for the evaluated transaction. Pass it as
	// ProcessRequest.TransactionID to re-evaluate the transaction later.
	TransactionID string `json:"transaction_id,omitempty"`

	// Allow means the payment can proceed.
	Allow        bool   `json:"allow"`
	AllowMessage string `json:"allow_message,omitempty"`
	// Challenge means the customer must complete an extra verification step
	// before the payment can proceed.
	Challenge        bool   `json:"challenge"`
	ChallengeMessage string `json:"challenge_message,omitempty"`
	// ChallengeType is the verification type configured on the rule. Its
	// values are defined in the rule, not by Aurora.
	ChallengeType string `json:"challenge_type,omitempty"`
	// Error means at least one rule failed to evaluate. See the type comment.
	Error        bool   `json:"error"`
	ErrorMessage string `json:"error_message,omitempty"`
	// ExecutionTime is how long the rule evaluation took on Aurora's side.
	ExecutionTime  time.Duration `json:"execution_time"`
	InspectMessage string        `json:"inspect_message,omitempty"`
	// NeedInspect means the payment is held for manual review.
	NeedInspect   bool   `json:"need_inspect"`
	RejectMessage string `json:"reject_message,omitempty"`
	// Rejected means the payment must be blocked.
	Rejected bool `json:"rejected"`
	// Score is the sum of the scores added by the evaluated rules. It is
	// informational: the flags above carry the decision.
	Score int `json:"score"`
}

// CallbackTransactionRequest is the input to CallbackService.Transaction.
type CallbackTransactionRequest struct {
	// Message describes the outcome. Required.
	Message string `json:"message"`
	// PaymentID identifies the payment. It must equal the common.payment_id
	// sent when the transaction was processed, and must match exactly one
	// transaction in your organization.
	PaymentID string `json:"payment_id"`
	// Status is CallbackStatusSuccess or CallbackStatusFailed.
	Status string `json:"status"`
}

// CallbackTransactionResponse is the acknowledgement Aurora returns for a
// recorded outcome.
type CallbackTransactionResponse struct {
	Message string `json:"message"`
}

// ErrorResponse is the JSON body Aurora sends with an error status. Depending
// on the endpoint, the text is under the "error" key or the "message" key.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
