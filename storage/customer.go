package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

// Customer represents a customer of the time tracking human.
type Customer struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// NewCustomer instantiates a customer object
func NewCustomer(name string) *Customer {
	return &Customer{
		ID:   uuid.New(),
		Name: name,
	}
}

// SavePath returns the save path of a customer.
func (c *Customer) SavePath(root string) string {
	return filepath.Join(root, "customers", c.ID.String())
}

// EnsureFolder tries to create the customers folder or fails.
func (c *Customer) EnsureFolder(root string) (string, error) {
	customerSavePath := c.SavePath(root)
	err := os.MkdirAll(customerSavePath, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("could not create directory %s: %v", customerSavePath, err)
	}
	return customerSavePath, nil
}

// Save saves a customer metadata
func (c *Customer) Save(root string) error {
	customerSavePath, err := c.EnsureFolder(root)
	if err != nil {
		return fmt.Errorf("ensuring customer folder: %v", err)
	}
	customerMetadataPath := filepath.Join(customerSavePath, "metadata.json")
	f, err := os.Create(customerMetadataPath)
	if err != nil {
		return fmt.Errorf("could not create customer metadata file: %w", err)
	}
	defer f.Close()
	m := json.NewEncoder(f)
	m.SetIndent("", "  ")
	err = m.Encode(c)
	if err != nil {
		return fmt.Errorf("could not save customer metadata: %w", err)
	}
	return nil
}

// ErrNotFound should be returned when the requested customer cannot be found.
var ErrNotFound = errors.New("customer not found")

// LoadCustomer reads a customer from disk.
func LoadCustomer(root string, id uuid.UUID) (*Customer, error) {
	c := &Customer{ID: id}
	customerPath := c.SavePath(root)
	if _, err := os.Stat(customerPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("customer %s does not exist: %w", id, err)
	}
	customerMetadataPath := filepath.Join(customerPath, "metadata.json")
	f, err := os.Open(customerMetadataPath)
	if err != nil {
		return nil, fmt.Errorf("open customer metadata file: %w", err)
	}
	defer f.Close()
	m := json.NewDecoder(f)
	err = m.Decode(c)
	if err != nil {
		return nil, fmt.Errorf("decode customer metadata file: %w", err)
	}
	return c, nil
}

// LoadAllCustomers loads all customers for this system
func LoadAllCustomers(root string) ([]Customer, error) {
	customersFolder := filepath.Join(root, "customers")
	matches, err := filepath.Glob(filepath.Join(customersFolder, "*"))
	if err != nil {
		return nil, fmt.Errorf("glob customers: %w", err)
	}
	fmt.Printf("Found %d customers in %s\n", len(matches), filepath.Join(customersFolder, "*"))
	customers := make([]Customer, len(matches))
	for i, match := range matches {
		match = filepath.Base(match)

		parsed, err := uuid.Parse(match)
		if err != nil {
			return nil, fmt.Errorf("parse customer UUID: %w", err)
		}
		c, err := LoadCustomer(root, parsed)
		if err != nil {
			return nil, fmt.Errorf("loading customer %q: %w", parsed, err)
		}
		customers[i] = *c
	}
	return customers, nil
}
