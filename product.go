package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrDuplicateTraceCode = errors.New("trace code already exists")
	ErrProductNotFound    = errors.New("product not found")
)

var traceCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type Product struct {
	TraceCode          string `json:"trace_code"`
	ProductCode        string `json:"product_code"`
	Name               string `json:"name"`
	BatchCode          string `json:"batch_code"`
	ManufacturedAt     string `json:"manufactured_at"`
	ExpiresAt          string `json:"expires_at"`
	Origin             string `json:"origin"`
	VerificationStatus string `json:"verification_status"`
}

func (p Product) normalized() Product {
	p.TraceCode = strings.TrimSpace(p.TraceCode)
	p.ProductCode = strings.TrimSpace(p.ProductCode)
	p.Name = strings.TrimSpace(p.Name)
	p.BatchCode = strings.TrimSpace(p.BatchCode)
	p.ManufacturedAt = strings.TrimSpace(p.ManufacturedAt)
	p.ExpiresAt = strings.TrimSpace(p.ExpiresAt)
	p.Origin = strings.TrimSpace(p.Origin)
	p.VerificationStatus = strings.TrimSpace(p.VerificationStatus)
	return p
}

func (p Product) Validate() error {
	p = p.normalized()
	if !traceCodePattern.MatchString(p.TraceCode) {
		return fmt.Errorf("trace_code must contain only letters, numbers, hyphens, or underscores")
	}
	for field, value := range map[string]string{
		"product_code": p.ProductCode,
		"name":         p.Name,
		"batch_code":   p.BatchCode,
		"origin":       p.Origin,
	} {
		if value == "" {
			return fmt.Errorf("%s must not be empty", field)
		}
	}

	manufacturedAt, err := time.Parse(time.DateOnly, p.ManufacturedAt)
	if err != nil {
		return fmt.Errorf("manufactured_at must use YYYY-MM-DD")
	}
	expiresAt, err := time.Parse(time.DateOnly, p.ExpiresAt)
	if err != nil {
		return fmt.Errorf("expires_at must use YYYY-MM-DD")
	}
	if expiresAt.Before(manufacturedAt) {
		return fmt.Errorf("expires_at must not be before manufactured_at")
	}
	if p.VerificationStatus != "verified" && p.VerificationStatus != "unverified" {
		return fmt.Errorf("verification_status must be verified or unverified")
	}

	return nil
}
