package document

import (
	"errors"
	"testing"
)

func TestTransitionRules(t *testing.T) {
	cases := []struct {
		typ      DocumentType
		from, to DocumentStatus
		ok       bool
	}{
		{TypeInvoice, StatusDraft, StatusPending, true},
		{TypeInvoice, StatusDraft, StatusCompleted, false},
		{TypeInvoice, StatusPending, StatusCompleted, true},
		{TypeInvoice, StatusPending, StatusOverdue, true},
		{TypeInvoice, StatusOverdue, StatusCompleted, true},
		{TypeInvoice, StatusOverdue, StatusPending, false},
		{TypeInvoice, StatusCancelled, StatusPending, false},
		{TypeInvoice, StatusCompleted, StatusCancelled, false},
		{TypeInvoice, StatusCompleted, StatusCompleted, true},
		{TypeInvoice, StatusPending, StatusSent, false},
		{TypeInvoice, StatusPending, StatusConverted, false},
		{TypeQuotation, StatusPending, StatusSent, true},
		{TypeQuotation, StatusSent, StatusAccepted, true},
		{TypeQuotation, StatusAccepted, StatusCancelled, true},
		{TypeQuotation, StatusAccepted, StatusPending, false},
		{TypeQuotation, StatusRejected, StatusAccepted, false},
		{TypeQuotation, StatusExpired, StatusSent, false},
		{TypeQuotation, StatusConverted, StatusCancelled, false},
		{TypeQuotation, StatusPending, StatusConverted, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.typ, c.from, c.to); got != c.ok {
			t.Errorf("%s %s→%s = %v, want %v", c.typ, c.from, c.to, got, c.ok)
		}
	}
	err := CheckTransition(TypeInvoice, StatusCancelled, StatusPending)
	var te *TransitionError
	if !errors.Is(err, ErrInvalidTransition) || !errors.As(err, &te) || te.From != StatusCancelled {
		t.Fatalf("CheckTransition = %v", err)
	}
	for _, s := range []DocumentStatus{StatusCancelled, StatusRejected, StatusExpired, StatusConverted} {
		if CanBeSource(s) {
			t.Errorf("%s must not be a conversion/billing source", s)
		}
	}
	if !CanBeSource(StatusPending) || !CanBeSource(StatusCompleted) {
		t.Error("open/completed documents must remain valid sources")
	}
}
