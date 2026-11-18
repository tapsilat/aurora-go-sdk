package aurora

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestShippingAddressJSON(t *testing.T) {
	data, err := json.Marshal(Common{ShippingAddress: &ShippingAddress{Address: "Main St 12/3", City: "Istanbul", Country: "TR", ZipCode: "34000"}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded Common
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ShippingAddress == nil || decoded.ShippingAddress.Address != "Main St 12/3" || decoded.ShippingAddress.ZipCode != "34000" {
		t.Fatalf("lost shipping address: %s", data)
	}
}

func TestTransactionCustomerAgeYearsJSON(t *testing.T) {
	payload, err := json.Marshal(Transaction{Type: TransactionTypeTransfer, CustomerAgeYears: "29"})
	if err != nil {
		t.Fatalf("marshal transaction: %v", err)
	}

	if !strings.Contains(string(payload), `"customer_age_years":"29"`) {
		t.Fatalf("expected customer_age_years in payload, got %s", payload)
	}
}

func TestTransactionCustomerProfileRiskFieldsJSON(t *testing.T) {
	payload, err := json.Marshal(Transaction{
		Type:                    TransactionTypeTransfer,
		CustomerAccountType:     CustomerAccountTypeIndividual,
		FirstTransactionAgeDays: "30",
	})
	if err != nil {
		t.Fatalf("marshal transaction: %v", err)
	}

	for _, expected := range []string{
		`"customer_account_type":"individual"`,
		`"first_transaction_age_days":"30"`,
	} {
		if !strings.Contains(string(payload), expected) {
			t.Errorf("expected %s in payload, got %s", expected, payload)
		}
	}
}

func TestProcessRequestLifecycleRootTransactionIDJSON(t *testing.T) {
	payload, err := json.Marshal(ProcessRequest{
		RuleID:                     "1e7b3b7b-0b3b-4b7b-8b3b-0b3b7b0b3b7b",
		LifecycleRootTransactionID: "9c2f5a41-6b0e-4c8a-9f31-2d7b8e4a1c05",
		Transaction:                &Transaction{Type: TransactionTypeRemittance},
	})
	if err != nil {
		t.Fatalf("marshal process request: %v", err)
	}

	if !strings.Contains(string(payload), `"lifecycle_root_transaction_id":"9c2f5a41-6b0e-4c8a-9f31-2d7b8e4a1c05"`) {
		t.Fatalf("expected lifecycle_root_transaction_id in payload, got %s", payload)
	}
}

func TestProcessRequestLifecycleRootTransactionIDOmittedJSON(t *testing.T) {
	payload, err := json.Marshal(ProcessRequest{
		RuleID:      "1e7b3b7b-0b3b-4b7b-8b3b-0b3b7b0b3b7b",
		Transaction: &Transaction{Type: TransactionTypeRemittance},
	})
	if err != nil {
		t.Fatalf("marshal process request: %v", err)
	}

	if strings.Contains(string(payload), "lifecycle_root_transaction_id") {
		t.Fatalf("expected lifecycle_root_transaction_id to be omitted, got %s", payload)
	}
}

func TestTransactionClientIPLocationJSON(t *testing.T) {
	payload, err := json.Marshal(Transaction{
		Type:      TransactionTypeTransfer,
		IPCountry: "AF",
		IPCity:    "Adana",
		Common: &Common{
			LocationCountry:    "AF",
			LocationRegion:     "Adana",
			LocationRegionCode: "01",
			LocationCity:       "Adana",
			LocationSource:     "ip.tapsilat.dev",
			Latitude:           "37.0000",
			Longitude:          "35.3213",
		},
	})
	if err != nil {
		t.Fatalf("marshal transaction: %v", err)
	}

	for _, expected := range []string{
		`"ip_country":"AF"`,
		`"ip_city":"Adana"`,
		`"location_country":"AF"`,
		`"location_region":"Adana"`,
		`"location_region_code":"01"`,
		`"location_city":"Adana"`,
		`"location_source":"ip.tapsilat.dev"`,
		`"latitude":"37.0000"`,
		`"longitude":"35.3213"`,
	} {
		if !strings.Contains(string(payload), expected) {
			t.Errorf("expected %s in payload, got %s", expected, payload)
		}
	}
}

func TestTransactionBookingJSON(t *testing.T) {
	payload, err := json.Marshal(Transaction{Type: TransactionTypeCard, Booking: &Booking{
		BookingReference: "BK-2026-104233", BookingType: "hotel", BookingChannel: "web", PropertyCountry: "TR",
		CheckInDate: "2026-12-20", CheckOutDate: "2026-12-23", GuestName: "Ayse Yilmaz", GuestCount: "2",
		RoomCount: "1", IsRefundable: "true", BookingTotal: "12500.00",
	}})
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Booking json.RawMessage `json:"booking"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil {
		t.Fatal(err)
	}
	want := `{"booking_reference":"BK-2026-104233","booking_type":"hotel","booking_channel":"web","property_country":"TR","check_in_date":"2026-12-20","check_out_date":"2026-12-23","guest_name":"Ayse Yilmaz","guest_count":"2","room_count":"1","is_refundable":"true","booking_total":"12500.00"}`
	if string(wire.Booking) != want {
		t.Fatalf("booking = %s, want %s", wire.Booking, want)
	}

	payload, err = json.Marshal(Transaction{Type: TransactionTypeCard})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"booking"`) {
		t.Fatalf("a transaction without booking must omit the group: %s", payload)
	}
}
