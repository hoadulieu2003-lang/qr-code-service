package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type ProductStore struct {
	path     string
	mu       sync.RWMutex
	products map[string]Product
}

func NewProductStore(path string) (*ProductStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create product data directory: %w", err)
	}

	store := &ProductStore{path: path, products: make(map[string]Product)}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read product data: %w", err)
	}
	if len(data) == 0 {
		return store, nil
	}

	var products []Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, fmt.Errorf("decode product data: %w", err)
	}
	for _, product := range products {
		product = product.normalized()
		if err := product.Validate(); err != nil {
			return nil, fmt.Errorf("validate product data: %w", err)
		}
		if _, exists := store.products[product.TraceCode]; exists {
			return nil, fmt.Errorf("validate product data: duplicate trace code %q", product.TraceCode)
		}
		store.products[product.TraceCode] = product
	}

	return store, nil
}

func (s *ProductStore) Create(product Product) error {
	product = product.normalized()
	if err := product.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.products[product.TraceCode]; exists {
		return ErrDuplicateTraceCode
	}

	s.products[product.TraceCode] = product
	if err := s.writeLocked(); err != nil {
		delete(s.products, product.TraceCode)
		return err
	}
	return nil
}

func (s *ProductStore) Get(traceCode string) (Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	product, exists := s.products[traceCode]
	if !exists {
		return Product{}, ErrProductNotFound
	}
	return product, nil
}

func (s *ProductStore) writeLocked() error {
	products := make([]Product, 0, len(s.products))
	for _, product := range s.products {
		products = append(products, product)
	}
	sort.Slice(products, func(i, j int) bool {
		return products[i].TraceCode < products[j].TraceCode
	})

	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return fmt.Errorf("encode product data: %w", err)
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(filepath.Dir(s.path), "products-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary product data: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect temporary product data: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary product data: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary product data: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("replace product data: %w", err)
	}

	return nil
}
