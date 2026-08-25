package main

import "testing"

// This test fails if invalid trace codes or expiry dates can enter public
// traceability data and later produce an unusable or misleading QR page.
func TestProductValidateRejectsInvalidDatesAndTraceCode(t *testing.T) {
	product := validProduct()
	product.TraceCode = "bad code!"
	if err := product.Validate(); err == nil {
		t.Fatal("Validate() accepted a trace code containing spaces and punctuation")
	}

	product = validProduct()
	product.ExpiresAt = "2026-08-24"
	if err := product.Validate(); err == nil {
		t.Fatal("Validate() accepted an expiry date before manufacture")
	}
}

// This test fails if surrounding whitespace becomes part of an identifier or
// public field after a product is persisted.
func TestProductNormalizedTrimsPublicFields(t *testing.T) {
	product := validProduct()
	product.TraceCode = " SP-DEMO-001 "
	product.Name = " Cà phê rang xay Demo "

	normalized := product.normalized()
	if normalized.TraceCode != "SP-DEMO-001" || normalized.Name != "Cà phê rang xay Demo" {
		t.Fatalf("normalized = %#v", normalized)
	}
}

func validProduct() Product {
	return Product{
		TraceCode:          "SP-DEMO-001",
		ProductCode:        "SP-001",
		Name:               "Cà phê rang xay Demo",
		BatchCode:          "LO-2026-001",
		ManufacturedAt:     "2026-08-25",
		ExpiresAt:          "2027-08-25",
		Origin:             "Đắk Lắk, Việt Nam",
		VerificationStatus: "verified",
	}
}
