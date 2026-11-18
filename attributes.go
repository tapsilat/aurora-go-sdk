package aurora

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

// Limits Aurora applies to Transaction.Attributes.
const (
	MaxAttributes           = 50
	MaxAttributeValueLength = 512
)

var attributeKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// ValidAttributeKey reports whether key may be used in Transaction.Attributes.
func ValidAttributeKey(key string) bool {
	return attributeKey.MatchString(key)
}

func validateAttributes(attributes map[string]string) error {
	if len(attributes) > MaxAttributes {
		return &ValidationError{Field: "transaction.attributes", Message: fmt.Sprintf("at most %d attributes are allowed", MaxAttributes)}
	}
	for key, value := range attributes {
		if !ValidAttributeKey(key) {
			return &ValidationError{Field: "transaction.attributes", Message: fmt.Sprintf("invalid attribute key %q", key)}
		}
		if utf8.RuneCountInString(value) > MaxAttributeValueLength {
			return &ValidationError{Field: "transaction.attributes", Message: fmt.Sprintf("attribute %q exceeds %d characters", key, MaxAttributeValueLength)}
		}
	}
	return nil
}
