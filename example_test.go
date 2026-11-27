package aurora_test

// These examples appear in the package documentation. They are compiled by
// go test (so they cannot drift from the API) but not run, because they
// talk to a real Aurora host.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	aurora "github.com/tapsilat/aurora-go-sdk"
)

func ExampleNewClient() {
	client := aurora.NewClient("your-api-key",
		aurora.WithBaseURL("https://aurora.example.com"),
		aurora.WithTimeout(10*time.Second),
	)
	_ = client // create once and reuse; the client is safe for concurrent use
}

func ExampleProcessService_Execute() {
	client := aurora.NewClient("your-api-key", aurora.WithBaseURL("https://aurora.example.com"))

	resp, err := client.Process.Execute(context.Background(), &aurora.ProcessRequest{
		RulesetID: "your-ruleset-id",
		Transaction: &aurora.Transaction{
			Type: aurora.TransactionTypeCard,
			Common: &aurora.Common{
				Amount:    250,
				Currency:  "TRY",
				PaymentID: "pay_123",
				Email:     "customer@example.com",
				IPAddress: "203.0.113.10",
			},
			Card: &aurora.Card{BinNumber: "454360", LastFour: "0001", Holder: "Jane Doe"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	switch {
	case resp.Rejected:
		fmt.Println("rejected:", resp.RejectMessage)
	case resp.Challenge:
		fmt.Println("challenge:", resp.ChallengeType, resp.ChallengeMessage)
	case resp.NeedInspect:
		fmt.Println("hold for review:", resp.InspectMessage)
	case resp.Allow:
		fmt.Println("allowed")
	default:
		// Error with no other decision: treat as not allowed.
		fmt.Println("evaluation failed:", resp.ErrorMessage)
	}
}

func ExampleProcessService_Execute_reevaluate() {
	client := aurora.NewClient("your-api-key", aurora.WithBaseURL("https://aurora.example.com"))

	// Aurora evaluates the stored transaction; no transaction body is needed.
	resp, err := client.Process.Execute(context.Background(), &aurora.ProcessRequest{
		RulesetID:     "your-ruleset-id",
		TransactionID: "0b0c6a9e-8f1d-4d6e-9a43-0c8f7d2b1e55",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("allowed:", resp.Allow)
}

func ExampleCallbackService_Transaction() {
	client := aurora.NewClient("your-api-key", aurora.WithBaseURL("https://aurora.example.com"))

	_, err := client.Callback.Transaction(context.Background(), &aurora.CallbackTransactionRequest{
		PaymentID: "pay_123", // the common.payment_id sent with the process request
		Status:    aurora.CallbackStatusSuccess,
		Message:   "Payment captured",
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleAPIError() {
	client := aurora.NewClient("your-api-key", aurora.WithBaseURL("https://aurora.example.com"))

	_, err := client.Process.Execute(context.Background(), &aurora.ProcessRequest{
		RulesetID:   "your-ruleset-id",
		Transaction: &aurora.Transaction{Type: aurora.TransactionTypeTransfer},
	})

	var (
		vErr   *aurora.ValidationError
		apiErr *aurora.APIError
	)
	switch {
	case err == nil:
		return
	case errors.As(err, &vErr):
		fmt.Println("fix the request field:", vErr.Field) // nothing was sent
	case aurora.IsRateLimited(err):
		fmt.Println("rate limited; retry later")
	case aurora.IsUnauthorized(err):
		fmt.Println("check the API key")
	case errors.As(err, &apiErr):
		fmt.Println("aurora returned", apiErr.StatusCode, apiErr.Message)
	case errors.Is(err, context.DeadlineExceeded):
		fmt.Println("timed out")
	default:
		fmt.Println("network error:", err)
	}
}
