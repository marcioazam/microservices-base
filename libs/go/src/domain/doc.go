// Package domain provides domain primitives with built-in validation.
//
// Domain primitives are value objects that encapsulate validation rules
// and ensure that invalid values cannot be created. This package provides
// common domain types used across microservices:
//
//   - Email: RFC 5322 compliant email addresses
//   - UUID: RFC 4122 UUID v4 identifiers
//   - ULID: Universally Unique Lexicographically Sortable Identifiers
//   - Money: Monetary values with currency handling
//   - PhoneNumber: E.164 formatted phone numbers
//   - URL: Validated URLs with scheme restrictions
//   - Timestamp: ISO 8601 timestamps
//   - Duration: Human-readable duration parsing
//
// All types implement json.Marshaler and json.Unmarshaler for seamless
// JSON serialization. Validation errors are returned during construction,
// ensuring that once created, a domain primitive is always valid.
//
// Example usage:
//
//	email, err := domain.NewEmail("user@example.com")
//	if err != nil {
//	    // Handle validation error
//	}
//
//	uuid := domain.NewUUID()
//	ulid := domain.NewULID()
//
//	money, _ := domain.NewMoney(1000, domain.USD) // $10.00
//	total, _ := money.Add(domain.MustNewMoney(500, domain.USD))
package domain
