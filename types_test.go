package aurora

import "testing"

func TestTransferSubtypeConstants(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  string
		want string
	}{
		{"TransferTypeWalletToWallet", TransferTypeWalletToWallet, "wallet_to_wallet"},
		{"TransferTypeWalletToIBAN", TransferTypeWalletToIBAN, "wallet_to_iban"},
		{"TransferTypeIBANToWallet", TransferTypeIBANToWallet, "iban_to_wallet"},
		{"TransferTypeRemittance", TransferTypeRemittance, "remittance"},
	} {
		if tc.got != tc.want {
			t.Errorf("expected %s to be %q, got %q", tc.name, tc.want, tc.got)
		}
	}
}

func TestWithdrawalMethodConstants(t *testing.T) {
	if WithdrawalMethodMoneyTransferCollection != "money_transfer_collection" {
		t.Errorf("expected WithdrawalMethodMoneyTransferCollection to be %q, got %q",
			"money_transfer_collection", WithdrawalMethodMoneyTransferCollection)
	}
}
