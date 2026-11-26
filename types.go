package aurora

// Values for Transaction.Type. These are the only types Aurora accepts.
const (
	TransactionTypeCard           = "card"
	TransactionTypeTransfer       = "transfer"
	TransactionTypeRemittance     = "remittance"
	TransactionTypeAccountOpening = "account_opening"
	TransactionTypeAccountLogin   = "account_login"
	TransactionTypeAccountChange  = "account_change"
	TransactionTypeRefund         = "refund"
	TransactionTypeWithdrawal     = "withdrawal"
	TransactionTypeDeposit        = "deposit"
	TransactionTypeP2P            = "p2p"
	TransactionTypeCrypto         = "crypto"
	TransactionTypeBNPL           = "bnpl"
	TransactionTypeLoan           = "loan"
	TransactionTypeInvoice        = "invoice"
)

// Values for Transfer.TransferType.
const (
	TransferTypeWalletToWallet = "wallet_to_wallet"
	TransferTypeWalletToIBAN   = "wallet_to_iban"
	TransferTypeIBANToWallet   = "iban_to_wallet"
	TransferTypeRemittance     = "remittance"
)

// Values for Withdrawal.WithdrawalMethod.
const (
	WithdrawalMethodMoneyTransferCollection = "money_transfer_collection"
)

// Values for Transaction.CustomerAccountType.
const (
	CustomerAccountTypeIndividual   = "individual"
	CustomerAccountTypeOrganization = "organization"
)

// validTransactionTypes is the set of Transaction.Type values Aurora accepts.
// Keep it in sync with the constants above.
var validTransactionTypes = map[string]bool{
	TransactionTypeCard:           true,
	TransactionTypeTransfer:       true,
	TransactionTypeRemittance:     true,
	TransactionTypeAccountOpening: true,
	TransactionTypeAccountLogin:   true,
	TransactionTypeAccountChange:  true,
	TransactionTypeRefund:         true,
	TransactionTypeWithdrawal:     true,
	TransactionTypeDeposit:        true,
	TransactionTypeP2P:            true,
	TransactionTypeCrypto:         true,
	TransactionTypeBNPL:           true,
	TransactionTypeLoan:           true,
	TransactionTypeInvoice:        true,
}

// Values for CallbackTransactionRequest.Status.
const (
	CallbackStatusSuccess = "success"
	CallbackStatusFailed  = "failed"
)

// Transaction field names, for IntegrationSample.FixedFields and
// AbsentField.Field. Any transaction field name Aurora knows is accepted;
// these are the usual ones.
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

// Values for AbsentField.Reason: why a flow never carries a field.
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

// validAbsentReasons is the set of AbsentField.Reason values Aurora accepts.
// Keep it in sync with the constants above.
var validAbsentReasons = map[string]bool{
	AbsentNotAMember:   true,
	AbsentNoSession:    true,
	AbsentNotAnOrder:   true,
	AbsentNotCollected: true,
}

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

// Values for APIError.Code.
const (
	ErrorCodeInvalidRequest       = "invalid_request"
	ErrorCodeInvalidFraudType     = "invalid_fraud_type"
	ErrorCodeCurrencyMismatch     = "currency_mismatch"
	ErrorCodeForbiddenSource      = "forbidden_source"
	ErrorCodeTransactionNotFound  = "transaction_not_found"
	ErrorCodeAmbiguousTransaction = "ambiguous_transaction"
	ErrorCodeExternalIDConflict   = "external_id_bound_to_other_transaction"
)
