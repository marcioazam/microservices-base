package domain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

// Currency represents an ISO 4217 currency code.
type Currency string

// Common currency codes.
const (
	USD Currency = "USD"
	EUR Currency = "EUR"
	GBP Currency = "GBP"
	JPY Currency = "JPY"
	BRL Currency = "BRL"
	CNY Currency = "CNY"
	INR Currency = "INR"
	AUD Currency = "AUD"
	CAD Currency = "CAD"
	CHF Currency = "CHF"
)

// currencyDecimals maps currencies to their decimal places.
var currencyDecimals = map[Currency]int{
	USD: 2, EUR: 2, GBP: 2, JPY: 0, BRL: 2,
	CNY: 2, INR: 2, AUD: 2, CAD: 2, CHF: 2,
}

// Money represents a monetary value with currency.
type Money struct {
	amount   *big.Int // Amount in smallest unit (cents, etc.)
	currency Currency
}

// NewMoney creates a new Money from amount in smallest unit.
func NewMoney(amount int64, currency Currency) (Money, error) {
	if _, ok := currencyDecimals[currency]; !ok {
		return Money{}, fmt.Errorf("unsupported currency: %s", currency)
	}
	return Money{
		amount:   big.NewInt(amount),
		currency: currency,
	}, nil
}

// MustNewMoney creates a new Money, panicking on invalid input.
func MustNewMoney(amount int64, currency Currency) Money {
	m, err := NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

// NewMoneyFromFloat creates Money from a float value.
func NewMoneyFromFloat(amount float64, currency Currency) (Money, error) {
	decimals, ok := currencyDecimals[currency]
	if !ok {
		return Money{}, fmt.Errorf("unsupported currency: %s", currency)
	}
	multiplier := int64(1)
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}
	cents := int64(amount * float64(multiplier))
	return Money{
		amount:   big.NewInt(cents),
		currency: currency,
	}, nil
}

// Zero returns zero money for a currency.
func Zero(currency Currency) Money {
	return Money{amount: big.NewInt(0), currency: currency}
}

// Amount returns the amount in smallest unit.
func (m Money) Amount() int64 {
	if m.amount == nil {
		return 0
	}
	return m.amount.Int64()
}

// Currency returns the currency.
func (m Money) Currency() Currency {
	return m.currency
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.amount == nil || m.amount.Sign() == 0
}

// IsPositive returns true if the amount is positive.
func (m Money) IsPositive() bool {
	return m.amount != nil && m.amount.Sign() > 0
}

// IsNegative returns true if the amount is negative.
func (m Money) IsNegative() bool {
	return m.amount != nil && m.amount.Sign() < 0
}

// Add adds two Money values.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.currency, other.currency)
	}
	result := new(big.Int).Add(m.amount, other.amount)
	return Money{amount: result, currency: m.currency}, nil
}

// Subtract subtracts another Money value.
func (m Money) Subtract(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.currency, other.currency)
	}
	result := new(big.Int).Sub(m.amount, other.amount)
	return Money{amount: result, currency: m.currency}, nil
}

// Multiply multiplies by a scalar.
func (m Money) Multiply(factor int64) Money {
	result := new(big.Int).Mul(m.amount, big.NewInt(factor))
	return Money{amount: result, currency: m.currency}
}

// Divide divides by a scalar with rounding.
func (m Money) Divide(divisor int64) Money {
	result := new(big.Int).Div(m.amount, big.NewInt(divisor))
	return Money{amount: result, currency: m.currency}
}

// Equals checks if two Money values are equal.
func (m Money) Equals(other Money) bool {
	return m.currency == other.currency && m.amount.Cmp(other.amount) == 0
}

// Compare compares two Money values.
func (m Money) Compare(other Money) int {
	if m.currency != other.currency {
		return strings.Compare(string(m.currency), string(other.currency))
	}
	return m.amount.Cmp(other.amount)
}

// String returns a human-readable representation.
func (m Money) String() string {
	decimals := currencyDecimals[m.currency]
	if decimals == 0 {
		return fmt.Sprintf("%s %d", m.currency, m.amount.Int64())
	}
	divisor := int64(1)
	for i := 0; i < decimals; i++ {
		divisor *= 10
	}
	whole := m.amount.Int64() / divisor
	frac := m.amount.Int64() % divisor
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%s %d.%0*d", m.currency, whole, decimals, frac)
}

// moneyJSON is the JSON representation of Money.
type moneyJSON struct {
	Amount   int64    `json:"amount"`
	Currency Currency `json:"currency"`
}

// MarshalJSON implements json.Marshaler.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(moneyJSON{
		Amount:   m.Amount(),
		Currency: m.currency,
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *Money) UnmarshalJSON(data []byte) error {
	var j moneyJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	money, err := NewMoney(j.Amount, j.Currency)
	if err != nil {
		return err
	}
	*m = money
	return nil
}
