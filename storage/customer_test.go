package storage

import (
	"encoding/json"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"testing"
)

func TestCustomer_SavePath(t *testing.T) {
	tmpFolder := t.TempDir()
	tests := []struct {
		name     string
		customer Customer
		root     string
		want     string
	}{
		{
			name: "valid customer save path",
			customer: Customer{
				ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Name: "Test Customer",
			},
			root: tmpFolder,
			want: filepath.Join(tmpFolder, "customers", "123e4567-e89b-12d3-a456-426614174000"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.customer.SavePath(tt.root); got != tt.want {
				t.Errorf("Customer.SavePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomer_EnsureFolder(t *testing.T) {
	tmpFolder := t.TempDir()
	tests := []struct {
		name     string
		customer Customer
		root     string
		wantErr  bool
	}{
		{
			name: "valid customer ensure folder",
			customer: Customer{
				ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Name: "Test Customer",
			},
			root:    tmpFolder,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.customer.EnsureFolder(tt.root)
			if (err != nil) != tt.wantErr {
				t.Errorf("Customer.EnsureFolder() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Also check that the folder exists
			if _, err := os.Stat(tt.customer.SavePath(tt.root)); os.IsNotExist(err) {
				t.Errorf("Customer.EnsureFolder() did not create the folder")
			}
		})
	}
}

func TestCustomer_Save(t *testing.T) {
	tmpFolder := t.TempDir()
	tests := []struct {
		name         string
		customer     Customer
		root         string
		expectedJSON json.RawMessage
		wantErr      bool
	}{
		{
			name: "valid customer save",
			customer: Customer{
				ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
				Name: "Test Customer",
			},
			root: tmpFolder,
			expectedJSON: []byte(`{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "Test Customer"
}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.customer.Save(tt.root); (err != nil) != tt.wantErr {
				t.Errorf("Customer.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Also check that the file exists and has the expected content
			customerPath := filepath.Join(tt.customer.SavePath(tt.root), "metadata.json")
			f, err := os.Open(customerPath)
			if err != nil {
				t.Fatalf("Customer.Save() did not create the file: %v", err)
			}
			defer f.Close()
			var gotJSON json.RawMessage
			if err := json.NewDecoder(f).Decode(&gotJSON); err != nil {
				t.Fatalf("Customer.Save() could not decode the file: %v", err)
			}
			if string(gotJSON) != string(tt.expectedJSON) {
				t.Errorf("Customer.Save() = %s, want %s", string(gotJSON), string(tt.expectedJSON))
			}
		})
	}
}

func TestLoadCustomer(t *testing.T) {
	tmpFolder := t.TempDir()
	customer := Customer{
		ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Name: "Test Customer",
	}
	if err := customer.Save(tmpFolder); err != nil {
		t.Fatalf("Failed to save customer: %v", err)
	}

	tests := []struct {
		name    string
		root    string
		id      uuid.UUID
		want    *Customer
		wantErr bool
	}{
		{
			name:    "valid load customer",
			root:    tmpFolder,
			id:      customer.ID,
			want:    &customer,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadCustomer(tt.root, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadCustomer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && *got != *tt.want {
				t.Errorf("LoadCustomer() = %v, want %v", got, tt.want)
			}
			// also check that the loaded customer is the same as the saved one
			if *got != customer {
				t.Errorf("LoadCustomer() = %v, want %v", got, &customer)
			}
		})
	}
}

func TestLoadAllCustomers(t *testing.T) {
	tmpFolder := t.TempDir()
	customer1 := Customer{
		ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Name: "Test Customer 1",
	}
	customer2 := Customer{
		ID:   uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
		Name: "Test Customer 2",
	}
	if err := customer1.Save(tmpFolder); err != nil {
		t.Fatalf("Failed to save customer1: %v", err)
	}
	if err := customer2.Save(tmpFolder); err != nil {
		t.Fatalf("Failed to save customer2: %v", err)
	}

	tests := []struct {
		name    string
		root    string
		want    []Customer
		wantErr bool
	}{
		{
			name:    "valid load all customers",
			root:    tmpFolder,
			want:    []Customer{customer1, customer2},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadAllCustomers(tt.root)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadAllCustomers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equalCustomers(got, tt.want) {
				t.Errorf("LoadAllCustomers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func equalCustomers(a, b []Customer) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
