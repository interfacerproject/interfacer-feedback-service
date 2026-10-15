// Package authz is the feedback service's enforcement-point client of the
// cross-service Policy Decision Point. BuildRequest and Interpret are pure so
// the request shape and the fail-closed mapping are unit-testable without a
// live PDP. The action is fixed by the route; the PDP returns the proven
// subject (Person.id) the service writes under — never the self-asserted
// x-user-id header.
package authz

// Proof carries the exact signed bytes (base64) and their signature, so the
// PDP re-verifies the user's own signature (contract §4.3 property 2).
type Proof struct {
	Signature  string `json:"signature"`
	PayloadB64 string `json:"payload_b64"`
}

// Request is the contract request for an identity-proof (feedback) action: no
// binding, since feedback has no resource parent.
type Request struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
	Principal struct {
		Pk string `json:"pk"`
	} `json:"principal"`
	Proof Proof `json:"proof"`
}

// Response is the contract response. Subject is the proven Person.id, present
// only on an allow that resolved the key.
type Response struct {
	Decision      string         `json:"decision"`
	ReasonCode    string         `json:"reason_code"`
	Subject       string         `json:"subject"`
	Obligations   map[string]any `json:"obligations"`
	RequestID     string         `json:"request_id"`
	PolicyVersion string         `json:"policy_version"`
}

// Outcome is what the middleware enforces.
type Outcome struct {
	Allow      bool
	Status     int
	ReasonCode string
	Subject    string
}

// BuildRequest assembles the wire request. action is supplied by the route.
func BuildRequest(requestID, action, pk string, proof Proof) Request {
	var r Request
	r.RequestID = requestID
	r.Action = action
	r.Principal.Pk = pk
	r.Proof = proof
	return r
}

// Interpret maps a PDP response (or transport error) to an enforced outcome.
// Fail closed: only an "allow" that carries a proven subject proceeds.
func Interpret(resp Response, transportErr error) Outcome {
	if transportErr != nil {
		return Outcome{Allow: false, Status: 403, ReasonCode: "pdp_unreachable"}
	}
	if resp.Decision == "allow" {
		if resp.Subject == "" {
			return Outcome{Allow: false, Status: 403, ReasonCode: "no_subject"}
		}
		return Outcome{Allow: true, Status: 200, ReasonCode: resp.ReasonCode, Subject: resp.Subject}
	}
	reason := resp.ReasonCode
	if reason == "" {
		reason = "denied"
	}
	return Outcome{Allow: false, Status: 403, ReasonCode: reason}
}
