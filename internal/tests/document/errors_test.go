package document_test

import (
	"testing"

	"pos-backend/internal/modules/document"
)

func TestDocumentErrors_NotEmpty(t *testing.T) {
	errors := []error{
		document.ErrInvalidInput,
		document.ErrNotFound,
		document.ErrForbidden,
	}
	for _, e := range errors {
		if e == nil {
			t.Error("error sentinel must not be nil")
		}
		if e.Error() == "" {
			t.Errorf("error message must not be empty: %T", e)
		}
	}
}
