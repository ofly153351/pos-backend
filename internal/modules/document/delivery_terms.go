package document

import "time"

// calculateTermDates keeps quotation term arithmetic in one place. Nil values
// intentionally remain nil so legacy documents and blank form fields stay blank.
func calculateTermDates(documentDate time.Time, priceValidityDays, deliveryLeadTimeDays *int, poReceivedDate *time.Time) (validUntil, expectedDeliveryDate *time.Time) {
	if priceValidityDays != nil {
		valid := documentDate.AddDate(0, 0, *priceValidityDays)
		validUntil = &valid
	}
	if deliveryLeadTimeDays != nil && poReceivedDate != nil {
		expected := poReceivedDate.AddDate(0, 0, *deliveryLeadTimeDays)
		expectedDeliveryDate = &expected
	}
	return validUntil, expectedDeliveryDate
}
