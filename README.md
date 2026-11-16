# Aurora Go SDK

Official Go SDK for the [Aurora API](https://kints.dev) — a rule-based transaction processing engine.

## Installation

```bash
go get github.com/kintsdev/aurora-go-sdk
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	aurora "github.com/kintsdev/aurora-go-sdk"
)

func main() {
	client := aurora.NewClient("your-api-key",
		aurora.WithBaseURL("https://api.example.com"),
	)

	resp, err := client.Process.Execute(context.Background(), &aurora.ProcessRequest{
		RulesetID: "your-ruleset-id",
		Transaction: &aurora.Transaction{
			Type: aurora.TransactionTypeCard,
			Common: &aurora.Common{
				Amount:   250.00,
				Currency: "USD",
				Email:    "customer@example.com",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Allowed: %v | Rejected: %v | Score: %d\n", resp.Allow, resp.Rejected, resp.Score)
}
```

## Configuration

### Client Options

| Option | Description |
|---|---|
| `WithBaseURL(url)` | **Required.** Set the API host (e.g. `https://api.example.com`). The path `/api/v1` is appended automatically. Without it, every call returns `aurora.ErrMissingBaseURL`. |
| `WithHTTPClient(client)` | Provide a custom `*http.Client`. It is never modified; `nil` is ignored. |
| `WithTimeout(duration)` | Set the request timeout (default: 30s). With `WithHTTPClient`, it is applied to a copy of your client. Option order does not matter. |

Create one client and reuse it; it is safe for concurrent use.

### Authentication

The SDK sends the API key as a `Bearer` token in the `Authorization` header.

```go
client := aurora.NewClient("your-api-key")
```

## Process

Execute a rule or ruleset against a transaction using `client.Process.Execute()`.
Either `RuleID` or `RulesetID` is required, and a new transaction needs a valid `Type`.

### With a Single Rule

```go
resp, err := client.Process.Execute(ctx, &aurora.ProcessRequest{
    RuleID: "rule-id",
    Transaction: &aurora.Transaction{
        Type: aurora.TransactionTypeCard,
        Common: &aurora.Common{
            Amount:   100.00,
            Currency: "TRY",
            UserID:   "user-123",
        },
    },
})
```

### With a Ruleset

```go
resp, err := client.Process.Execute(ctx, &aurora.ProcessRequest{
    RulesetID: "ruleset-id",
    Transaction: &aurora.Transaction{
        Type: aurora.TransactionTypeTransfer,
        Common: &aurora.Common{
            Amount:   500.00,
            Currency: "EUR",
        },
        Transfer: &aurora.Transfer{
            PaymentMethod: "bank_deposit",
            SenderCountry: "DE",
        },
    },
})
```

### Re-evaluating a Stored Transaction

Pass the `TransactionID` returned by an earlier call. Aurora evaluates the
stored transaction, so `Transaction` can be omitted.

```go
resp, err := client.Process.Execute(ctx, &aurora.ProcessRequest{
    RulesetID:     "ruleset-id",
    TransactionID: previous.TransactionID,
})
```

### Full Example

```go
resp, err := client.Process.Execute(ctx, &aurora.ProcessRequest{
    RulesetID: "2a8c4d9e-1f2a-4b3c-9d8e-7f6a5b4c3d2e",
    Transaction: &aurora.Transaction{
        Type:            aurora.TransactionTypeCard,
        TransactionType: "purchase",
        AccountAgeDays:  "180",
        CustomerAgeYears: "29",
        IsNewDevice:     "true",
        IsNewIP:         "false",
        IsUnusualLocation: "true",
        Common: &aurora.Common{
            Amount:      99.99,
            Currency:    "USD",
            Category:    "ecommerce",
            Country:     "US",
            Email:       "john@doe.com",
            IPAddress:   "127.0.0.1",
            UserID:      "user_1234567890",
            MerchantID:  "merch_1234567890",
            PaymentID:   "pay_1234567890",
            ReferenceID: "order_1234567890",
        },
        Card: &aurora.Card{
            BinNumber: "123456",
            Holder:    "John Doe",
            Issuer:    "Bank of America",
            LastFour:  "1234",
            MCC:       "5411",
            Network:   "Visa",
            Token:     "card_tok_1234567890",
            Type:      "credit",
        },
        Transfer: &aurora.Transfer{
            TransferType:    "wallet_to_wallet",
            TransferPurpose: "family_support",
            SenderName:      "John Doe",
            SenderCountry:   "US",
            ReceiverName:    "Jane Smith",
            ReceiverCountry: "GB",
        },
    },
})
```

### Response

```go
type ProcessResponse struct {
    TransactionID    string        // Aurora transaction ID; use it to re-evaluate
    Allow            bool          // transaction is allowed
    AllowMessage     string        // message when allowed
    Rejected         bool          // transaction is rejected
    RejectMessage    string        // reason for rejection
    Challenge        bool          // customer must pass an extra check (e.g. OTP)
    ChallengeMessage string        // reason for the challenge
    ChallengeType    string        // kind of challenge requested
    NeedInspect      bool          // transaction needs manual review
    InspectMessage   string        // reason for inspection
    Score            int           // risk score
    Warnings         []RuleWarning // non-blocking findings
    Error            bool          // at least one active rule failed to evaluate
    ErrorMessage     string        // error details
    ExecutionTime    time.Duration // rule execution duration
}
```

Normally exactly one of `Allow`, `Rejected`, `Challenge` and `NeedInspect` is
true. `Error` can be set together with a `Rejected`, `Challenge` or
`NeedInspect` decision; treat a response with `Error` and no other decision as
not allowed.

### Transaction Fields

The `Transaction` struct uses nested sub-structs to organize fields by category. Only populate the sections relevant to your use case — all sub-struct pointers and fields use `omitempty`.

#### Top-Level Scalar Fields

| Field | JSON Key | Description |
|---|---|---|
| `Type` | `type` | Transaction type (e.g. `card`) |
| `TransactionType` | `transaction_type` | Transaction action (e.g. `purchase`) |
| `AccountAgeDays` | `account_age_days` | Account age in days |
| `CustomerAgeYears` | `customer_age_years` | Customer age in completed years, calculated from date of birth |
| `DeclinedCount` | `declined_count` | Number of declined transactions |
| `IncomeMultiplier` | `income_multiplier` | Ratio of amount to registered income |
| `IPCountry` | `ip_country` | ISO country code resolved from the transaction IP |
| `IPCity` | `ip_city` | City resolved from the transaction IP |
| `IsFirstTransfer` | `is_first_transfer` | Whether this is the first transfer |
| `IsNewDevice` | `is_new_device` | Whether a new device is used |
| `IsNewIP` | `is_new_ip` | Whether a new IP is used |
| `IsUnusualLocation` | `is_unusual_location` | Whether location is unusual |
| `PasswordChangedRecently` | `password_changed_recently` | Recent password change flag |
| `ProfileCompletion` | `profile_completion` | Profile completion percentage |
| `RefundCount` | `refund_count` | Number of refunds |
| `RefundRatio` | `refund_ratio` | Refund to transaction ratio |
| `RegisteredIncome` | `registered_income` | User's registered income |
| `TotalAmount24h` | `total_amount_24h` | Total transaction amount in 24h |
| `TotalAmount7d` | `total_amount_7d` | Total transaction amount in 7 days |
| `TransactionHour` | `transaction_hour` | Hour of the transaction |
| `TransferCount24h` | `transfer_count_24h` | Number of transfers in 24h |
| `UniqueRecipients` | `unique_recipients` | Number of unique recipients |

#### Nested Sub-Structs

| Sub-Struct | JSON Key | Fields |
|---|---|---|
| `Common` | `common` | `Amount`, `Balance`, `BrowserAgent`, `Category`, `ConnectionType`, `Country`, `Currency`, `Date`, `Description`, `DeviceFingerprint`, `DeviceID`, `Email`, `FirstName`, `IdentityNumber`, `IPAddress`, `LastLoginIP`, `LastLoginTime`, `LastName`, `Latitude`, `LocationCity`, `LocationCountry`, `LocationRegion`, `LocationRegionCode`, `LocationSource`, `Longitude`, `MerchantID`, `MerchantName`, `PaymentID`, `Phone`, `ReferenceID`, `UserID` |
| `Card` | `card` | `BinNumber`, `Holder`, `Issuer`, `LastFour`, `MCC`, `Network`, `Token`, `Type` |
| `Transfer` | `transfer` | `ExchangeRate`, `PaymentMethod`, `ReceiverAddress`, `ReceiverCountry`, `ReceiverIBAN`, `ReceiverIdentityPassport`, `ReceiverName`, `ReceiverSurname`, `ReceiverWalletID`, `Relationship`, `SenderAddress`, `SenderCountry`, `SenderIBAN`, `SenderIdentityPassport`, `SenderName`, `SenderSurname`, `SenderWalletID`, `SourceOfFunds`, `TargetCurrency`, `TransferPurpose`, `TransferType` |
| `AccountChange` | `account_change` | `ChangeType`, `PreviousValueHash`, `TimeSinceLastChange`, `VerificationMethod` |
| `AccountLogin` | `account_login` | `FailedAttempts`, `LoginMethod`, `LoginStatus`, `MFAMethod`, `MFAUsed`, `SessionID` |
| `AccountOpening` | `account_opening` | `DocumentType`, `IdentityVerificationStatus`, `RegistrationMethod` |
| `BNPL` | `bnpl` | `BNPLProvider`, `InstallmentCount`, `OutstandingBNPLAmount` |
| `Crypto` | `crypto` | `CryptoAmount`, `CryptoCurrency`, `ExchangeName`, `IsSmartContract`, `WalletAddress` |
| `Deposit` | `deposit` | `CheckNumber`, `DepositMethod`, `DepositSource`, `IsRemoteDeposit` |
| `Invoice` | `invoice` | `InvoiceDueDate`, `InvoiceNumber`, `IsRecurringVendor`, `PurchaseOrderID`, `VendorID`, `VendorName` |
| `Loan` | `loan` | `CreditScore`, `DebtToIncomeRatio`, `EmploymentStatus`, `LoanAmount`, `LoanTerm`, `LoanType` |
| `P2P` | `p2p` | `PaymentNote`, `Platform`, `RecipientAccountAge` |
| `Refund` | `refund` | `DaysSincePurchase`, `OriginalTransactionID`, `RefundMethod`, `RefundReason` |
| `Withdrawal` | `withdrawal` | `AccountType`, `ATMID`, `DailyWithdrawalCount`, `WithdrawalMethod` |

## Error Handling

| Error | When |
|---|---|
| `*aurora.ValidationError` | The request is invalid; nothing was sent. `Field` names the problem. |
| `*aurora.APIError` | Aurora returned a status of 400 or above. `Message` is the server's message, `Body` the raw response. |
| `aurora.ErrMissingBaseURL` | The client was created without `WithBaseURL`. |
| other | Network failure, timeout or context cancellation (check with `errors.Is(err, context.DeadlineExceeded)`). |

```go
resp, err := client.Process.Execute(ctx, req)
if err != nil {
    var apiErr *aurora.APIError
    switch {
    case aurora.IsRateLimited(err):
        // back off and retry later
    case aurora.IsUnauthorized(err):
        // check the API key
    case errors.As(err, &apiErr):
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
    }
    log.Fatal(err)
}
```

`aurora.IsNotFound(err)` is also available.

## Callback

Report the final outcome of a payment using `client.Callback.Transaction()`.
`PaymentID` must match the `common.payment_id` sent with the process request.
Sending the same status again is safe.

```go
resp, err := client.Callback.Transaction(ctx, &aurora.CallbackTransactionRequest{
    Message:   "Payment completed",
    PaymentID: "pay_1234567890",
    Status:    aurora.CallbackStatusSuccess,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Message: %s\n", resp.Message)
```

### Request

| Field | JSON Key | Description |
|---|---|---|
| `Message` | `message` | Outcome description. Required. |
| `PaymentID` | `payment_id` | The `common.payment_id` of the processed transaction. Required. |
| `Status` | `status` | `aurora.CallbackStatusSuccess` (`success`) or `aurora.CallbackStatusFailed` (`failed`). Required. |

### Response

```go
type CallbackTransactionResponse struct {
    Message string // response message
}
```

## Integration Samples

Tell Aurora what your integration sends, so its rule self-test can show, before any real transaction arrives, on which of your transaction flows each rule can act and why not on the others. Build the samples with the same code that builds your real `Process` requests.

```go
_, err := client.Integration.PublishSamples(ctx, &aurora.PublishSamplesRequest{
    Source: "admin",
    Samples: []aurora.IntegrationSample{{
        Flow:        "bank_withdraw",
        Title:       "Banka çekimi",
        // Fields with the same value on every transaction of this flow.
        FixedFields: []string{aurora.FieldTransactionType, aurora.FieldPaymentMethod, aurora.FieldCurrency},
        Transaction: buildBankWithdrawTransaction(sampleInput),
    }},
})
```

A publish replaces every sample the source published before; an empty list removes them. Fields left empty in a sample are taken to be never sent, fields with a value that are not in `FixedFields` are taken to vary.

When a flow never carries a field by its nature, say so with `Absent`. Aurora then reports that a rule needing it is not meant for this flow, rather than that the data is missing:

```go
aurora.IntegrationSample{
    Flow: "partner_reference_issue",
    // The sender is a partner institution: no member account.
    Absent: []aurora.AbsentField{
        {Field: aurora.FieldAccountAgeDays, Reason: aurora.AbsentNotAMember},
        {Field: aurora.FieldBillingName, Reason: aurora.AbsentNotAnOrder},
    },
    Transaction: buildPartnerIssueTransaction(sampleInput),
}
```

Reasons: `AbsentNotAMember` (no member account), `AbsentNoSession` (no user session, so no IP or device), `AbsentNotAnOrder` (no billing or shipping) and `AbsentNotCollected` (the flow does not collect it). Absent fields must be empty in the sample.

## License

MIT
