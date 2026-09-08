package main

import (
	"errors"
	"path/filepath"
	"testing"
)

// This test fails if a created product disappears after process restart or if
// a repeated trace code silently overwrites an existing product.
func TestProductStorePersistsAndRejectsDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "products.json")
	store, err := NewProductStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(validProduct()); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(validProduct()); !errors.Is(err, ErrDuplicateTraceCode) {
		t.Fatalf("duplicate create error = %v, want ErrDuplicateTraceCode", err)
	}

	reloaded, err := NewProductStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reloaded.Get("SP-DEMO-001")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Cà phê rang xay Demo" {
		t.Fatalf("reloaded product name = %q, want Cà phê rang xay Demo", got.Name)
	}
}

// This test fails if an unknown public trace code is represented as a zero
// product rather than an explicit not-found error.
func TestProductStoreReturnsNotFound(t *testing.T) {
	store, err := NewProductStore(filepath.Join(t.TempDir(), "products.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Get("SP-MISSING-001")
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("Get() error = %v, want ErrProductNotFound", err)
	}
}
