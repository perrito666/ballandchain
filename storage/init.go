package storage

import (
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

var customerFromID map[uuid.UUID]Customer
var taskFromID map[uuid.UUID]map[uuid.UUID]Task // this does not take in account possible clashes
// var taskIndex // coming soon
var defaultRoot string

func init() {
	rootFolder := os.Getenv("BAC_ROOT_FOLDER")
	var err error
	if rootFolder == "" {
		rootFolder, err = os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		rootFolder = filepath.Join(rootFolder, ".ballandchain")
	}
	defaultRoot = rootFolder
	var customers []Customer
	customers, err = LoadAllCustomers(defaultRoot)
	if err != nil {
		panic(err)
	}
	customerFromID = make(map[uuid.UUID]Customer, len(customers))
	taskFromID = make(map[uuid.UUID]map[uuid.UUID]Task, len(customers))
	for _, customer := range customers {
		// always populate customerFromID before loading tasks for that customer
		customerFromID[customer.ID] = customer
		cTasks, err := LoadTasks(defaultRoot, &customer)
		if err != nil {
			panic(err)
		}
		taskFromIDC := make(map[uuid.UUID]Task, len(cTasks.Tasks))
		for _, t := range cTasks.Tasks {
			taskFromIDC[t.ID] = *t
		}
		taskFromID[customer.ID] = taskFromIDC
	}
}
