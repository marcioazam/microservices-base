package crypto

import (
	"bytes"
	"encoding/json"
	"sort"
	"time"
)

// SignedDecision represents a signed authorization decision.
type SignedDecision struct {
	DecisionID string `json:"decision_id"`
	Timestamp  int64  `json:"timestamp"`
	SubjectID  string `json:"subject_id"`
	ResourceID string `json:"resource_id"`
	Action     string `json:"action"`
	Allowed    bool   `json:"allowed"`
	PolicyName string `json:"policy_name"`
	Signature  []byte `json:"signature,omitempty"`
	KeyID      KeyID  `json:"key_id,omitempty"`
}

// NewSignedDecision creates a new signed decision from authorization result.
func NewSignedDecision(decisionID, subjectID, resourceID, action, policyName string, allowed bool) *SignedDecision {
	return &SignedDecision{
		DecisionID: decisionID,
		Timestamp:  time.Now().Unix(),
		SubjectID:  subjectID,
		ResourceID: resourceID,
		Action:     action,
		Allowed:    allowed,
		PolicyName: policyName,
	}
}

// BuildSignaturePayload creates the canonical payload for signing.
// The payload is deterministic and includes all required fields.
func (d *SignedDecision) BuildSignaturePayload() []byte {
	// Create a map with all fields that should be signed
	payload := map[string]interface{}{
		"decision_id": d.DecisionID,
		"timestamp":   d.Timestamp,
		"subject_id":  d.SubjectID,
		"resource_id": d.ResourceID,
		"action":      d.Action,
		"allowed":     d.Allowed,
		"policy_name": d.PolicyName,
	}

	// Use canonical JSON encoding (sorted keys)
	return canonicalJSON(payload)
}

// HasAllRequiredFields returns true if all required fields are present.
func (d *SignedDecision) HasAllRequiredFields() bool {
	return d.DecisionID != "" &&
		d.Timestamp > 0 &&
		d.SubjectID != "" &&
		d.ResourceID != "" &&
		d.Action != "" &&
		d.PolicyName != ""
}

// IsSigned returns true if the decision has a signature.
func (d *SignedDecision) IsSigned() bool {
	return len(d.Signature) > 0
}

// Clone creates a deep copy of the signed decision.
func (d *SignedDecision) Clone() *SignedDecision {
	clone := *d
	if d.Signature != nil {
		clone.Signature = make([]byte, len(d.Signature))
		copy(clone.Signature, d.Signature)
	}
	return &clone
}

// canonicalJSON produces deterministic JSON output with sorted keys.
func canonicalJSON(v interface{}) []byte {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	// For maps, we need to sort keys
	if m, ok := v.(map[string]interface{}); ok {
		buf.WriteByte('{')
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			// Write key
			keyBytes, _ := json.Marshal(k)
			buf.Write(keyBytes)
			buf.WriteByte(':')
			// Write value
			valBytes, _ := json.Marshal(m[k])
			buf.Write(valBytes)
		}
		buf.WriteByte('}')
		return buf.Bytes()
	}

	encoder.Encode(v)
	return bytes.TrimSpace(buf.Bytes())
}
