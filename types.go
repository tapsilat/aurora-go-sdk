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
