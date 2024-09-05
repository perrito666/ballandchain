package storage

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

// Task represents a (potentially) recurring task for a customer
type Task struct {
	ID         uuid.UUID `json:"id"`
	Customer   *Customer `json:"customer"`
	ExternalID string    `json:"external_id"` // think jira PRJ-#### or similar
	Name       string    `json:"name"`
}

// MarshalJSON method for Task to be able to serialize customer
func (t *Task) MarshalJSON() ([]byte, error) {
	alias := &struct {
		CustomerID uuid.UUID `json:"customer"`
		*Task
	}{
		Task: t,
	}

	if t.Customer != nil {
		alias.CustomerID = t.Customer.ID
	}

	return json.MarshalIndent(alias, "", "  ")
}

// UnmarshalJSON method for Task to ble able to de-serialize Customer
func (t *Task) UnmarshalJSON(data []byte) error {
	aux := &struct {
		CustomerID uuid.UUID `json:"customer"`
		*Task
	}{
		Task: t,
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if aux.CustomerID == uuid.Nil {
		return fmt.Errorf("invalid task customer ID")
	}
	customer, ok := customerFromID[aux.CustomerID]
	if !ok {
		return fmt.Errorf("customer ID $%s of task is non existent: %w", aux.CustomerID.String(), ErrNotFound)
	}
	t.Customer = &customer

	return nil
}

// EntriesSavePath returns the entries save path for this task's customer
func (t *Task) EntriesSavePath(root string) string {
	return filepath.Join(root, "tasks", t.Customer.ID.String())
}

// EnsureTaskEntriesFolder creates the entries folder if it does not exist and returns it.
func (t *Task) EnsureTaskEntriesFolder(root string) (string, error) {
	entriesPath := t.EntriesSavePath(root)
	if err := os.MkdirAll(entriesPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create task entries folder %s: %w", entriesPath, err)
	}
	return entriesPath, nil
}

type CustomerTasks struct {
	Customer *Customer `json:"customer"`
	Tasks    []*Task   `json:"tasks"`
}

// Save will persist the customer tasks
func (c *CustomerTasks) Save(root string) error {
	customerSavePath, err := c.Customer.EnsureFolder(root)
	if err != nil {
		return fmt.Errorf("ensuring customer folder exist: %w", err)
	}
	tasksSavePath := filepath.Join(customerSavePath, "tasks.json")
	f, err := os.Create(tasksSavePath)
	if err != nil {
		return fmt.Errorf("create or truncate tasks file: %w", err)
	}
	defer f.Close()
	m := json.NewEncoder(f)
	m.SetIndent("", "  ")
	err = m.Encode(c.Tasks)
	if err != nil {
		return fmt.Errorf("encode tasks file: %w", err)
	}
	return nil
}

// LoadTasks Reads tasks for a given customer.
func LoadTasks(root string, c *Customer) (*CustomerTasks, error) {
	if c == nil {
		return nil, fmt.Errorf("LoadTasks: customer must not be nil")
	}
	customerSavePath := filepath.Join(root, "customers", c.ID.String())
	tasksSavePath := filepath.Join(customerSavePath, "tasks.json")
	if _, err := os.Stat(tasksSavePath); os.IsNotExist(err) {
		return &CustomerTasks{
			Customer: c,
			Tasks:    []*Task{},
		}, nil
	}
	f, err := os.Open(tasksSavePath)
	if err != nil {
		return nil, fmt.Errorf("open tasks file: %w", err)
	}
	defer f.Close()
	m := json.NewDecoder(f)
	ct := &CustomerTasks{}
	err = m.Decode(ct)
	if err != nil {
		return nil, fmt.Errorf("decode tasks file: %w", err)
	}
	return ct, nil
}
