package storage

import (
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

var customerFromID map[uuid.UUID]Customer
var taskFromID map[uuid.UUID]map[uuid.UUID]Task // this does not take in account possible clashes
// index with this https://github.com/blevesearch/bleve
var taskIndex map[uuid.UUID]bleve.Index // coming soon
var defaultRoot string

func init() {
	// FIXME: make all these into encapsulated functions so they can be called from tests too
	rootFolder := os.Getenv("BAC_ROOT_FOLDER")
	var err error
	if rootFolder == "" {
		rootFolder, err = os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		rootFolder = filepath.Join(rootFolder, ".ballandchain")
	}
	err = initForRoot(rootFolder)
	if err != nil {
		panic(err)
	}
}

// initForRoot initializes the storage package with a custom root folder, it is mostly to encapsulate the init function for testing
func initForRoot(root string) error {
	defaultRoot = root
	var customers []Customer
	var err error
	customers, err = LoadAllCustomers(defaultRoot)
	if err != nil {
		return fmt.Errorf("loading customers: %w", err)
	}
	customerFromID = make(map[uuid.UUID]Customer, len(customers))
	taskFromID = make(map[uuid.UUID]map[uuid.UUID]Task, len(customers))
	taskIndex = make(map[uuid.UUID]bleve.Index, len(customers))
	for _, customer := range customers {
		// always populate customerFromID before loading tasks for that customer
		customerFromID[customer.ID] = customer
		cTasks, err := LoadTasks(defaultRoot, &customer)
		if err != nil {
			return fmt.Errorf("loading tasks for customer %s: %w", customer.Name, err)
		}
		taskFromIDC := make(map[uuid.UUID]Task, len(cTasks.Tasks))
		customerIndex := bleve.NewIndexMapping()
		index, err := bleve.New("tasks_"+customer.ID.String(), customerIndex)
		if err != nil {
			return fmt.Errorf("creating bleve index for customer %s: %w", customer.Name, err)
		}
		taskIndex[customer.ID] = index
		for _, t := range cTasks.Tasks {
			err = index.Index(t.Name, t)
			if err != nil {
				return fmt.Errorf("indexing task %s for customer %s: %w", t.Name, customer.Name, err)
			}
			taskFromIDC[t.ID] = *t
		}
		taskFromID[customer.ID] = taskFromIDC
	}
	return nil
}
