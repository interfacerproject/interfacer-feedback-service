package authz

import (
	"errors"
	"testing"
)

func TestBuildRequestFixesActionAndCarriesProof(t *testing.T) {
	req := BuildRequest("rid-1", "feedback.review.create", "pkAlice",
		Proof{Signature: "SIG", PayloadB64: "e30="})
	if req.Action != "feedback.review.create" {
		t.Fatalf("action: got %q", req.Action)
	}
	if req.Principal.Pk != "pkAlice" {
		t.Fatalf("pk: got %q", req.Principal.Pk)
	}
	if req.Proof.Signature != "SIG" || req.Proof.PayloadB64 != "e30=" {
		t.Fatalf("proof not carried: %+v", req.Proof)
	}
	if req.RequestID != "rid-1" {
		t.Fatalf("request id: got %q", req.RequestID)
	}
}

func TestInterpretAllowWithSubjectProceeds(t *testing.T) {
	out := Interpret(Response{Decision: "allow", ReasonCode: "ok", Subject: "01PERSONALICE"}, nil)
	if !out.Allow || out.Status != 200 || out.Subject != "01PERSONALICE" {
		t.Fatalf("allow+subject must proceed with subject: %+v", out)
	}
}

func TestInterpretAllowWithEmptySubjectFailsClosed(t *testing.T) {
	out := Interpret(Response{Decision: "allow", ReasonCode: "ok", Subject: ""}, nil)
	if out.Allow || out.Status != 403 {
		t.Fatalf("an allow without a proven subject must fail closed: %+v", out)
	}
}

func TestInterpretDenyIs403(t *testing.T) {
	out := Interpret(Response{Decision: "deny", ReasonCode: "principal_not_resolved"}, nil)
	if out.Allow || out.Status != 403 || out.ReasonCode != "principal_not_resolved" {
		t.Fatalf("deny: %+v", out)
	}
}

func TestInterpretTransportErrorFailsClosed(t *testing.T) {
	out := Interpret(Response{}, errors.New("connection refused"))
	if out.Allow || out.Status != 403 || out.ReasonCode != "pdp_unreachable" {
		t.Fatalf("transport error must fail closed: %+v", out)
	}
}
