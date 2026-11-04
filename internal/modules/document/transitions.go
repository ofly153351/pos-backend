package document

import (
	"errors"
	"fmt"
)

// Owner-approved lifecycle (B06, 2026-11-04). One table drives single updates,
// bulk updates and conversion eligibility, so they can never disagree.
const (
	StatusSent      DocumentStatus = "SENT"
	StatusAccepted  DocumentStatus = "ACCEPTED"
	StatusRejected  DocumentStatus = "REJECTED"
	StatusExpired   DocumentStatus = "EXPIRED"
	StatusConverted DocumentStatus = "CONVERTED"
)

var (
	// ErrInvalidTransition is returned for a status change the rules forbid.
	ErrInvalidTransition = errors.New("invalid status transition")
	// ErrSourceNotEligible is returned when a cancelled/closed document is used
	// as the source of a conversion or bill (B07/B12).
	ErrSourceNotEligible = errors.New("source document is not eligible")
)

// AlreadyConvertedError names the existing active target (B07/B12) so the
// client can link to it and explain the 409.
type AlreadyConvertedError struct {
	DocumentNo string
}

func (e *AlreadyConvertedError) Error() string {
	return "document already converted or billed: " + e.DocumentNo
}

func (e *AlreadyConvertedError) Unwrap() error { return ErrAlreadyConverted }

// TransitionError carries the rejected pair for structured 409 responses.
type TransitionError struct {
	From, To DocumentStatus
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("cannot change status from %s to %s", e.From, e.To)
}

func (e *TransitionError) Unwrap() error { return ErrInvalidTransition }

var generalTransitions = map[DocumentStatus][]DocumentStatus{
	StatusDraft:   {StatusPending, StatusCancelled},
	StatusPending: {StatusOverdue, StatusCompleted, StatusCancelled},
	StatusOverdue: {StatusCompleted, StatusCancelled},
}

// Quotations additionally use the sales-negotiation states. CONVERTED is set
// only by the conversion flow, never through a status update.
var quotationTransitions = map[DocumentStatus][]DocumentStatus{
	StatusDraft:    {StatusSent},
	StatusPending:  {StatusSent, StatusAccepted, StatusRejected, StatusExpired},
	StatusSent:     {StatusAccepted, StatusRejected, StatusExpired, StatusCancelled},
	StatusAccepted: {StatusCancelled},
}

// IsTerminal reports whether no further manual status change is allowed.
func IsTerminal(status DocumentStatus) bool {
	switch status {
	case StatusCompleted, StatusCancelled, StatusConverted, StatusRejected, StatusExpired:
		return true
	}
	return false
}

// CanTransition reports whether a manual status update from -> to is allowed
// for the document type. Re-applying the current status is a no-op and allowed.
func CanTransition(docType DocumentType, from, to DocumentStatus) bool {
	if from == to {
		return true
	}
	if to == StatusConverted || IsTerminal(from) {
		return false
	}
	if contains(generalTransitions[from], to) {
		return true
	}
	return docType == TypeQuotation && contains(quotationTransitions[from], to)
}

// CheckTransition returns a *TransitionError when the change is not allowed.
func CheckTransition(docType DocumentType, from, to DocumentStatus) error {
	if CanTransition(docType, from, to) {
		return nil
	}
	return &TransitionError{From: from, To: to}
}

// CanBeSource reports whether a document may be converted or billed (B07/B12):
// cancelled, rejected, expired and already-converted documents are closed.
func CanBeSource(status DocumentStatus) bool {
	switch status {
	case StatusCancelled, StatusRejected, StatusExpired, StatusConverted:
		return false
	}
	return true
}

func contains(list []DocumentStatus, s DocumentStatus) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
