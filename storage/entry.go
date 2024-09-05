package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Entry represents a unit of work on a task
type Entry struct {
	ID      uuid.UUID  `json:"id"`
	Task    *Task      `json:"task"`
	Comment string     `json:"comment"`
	StartTS time.Time  `json:"start_ts"`
	EndTs   *time.Time `json:"end_ts,omitempty"`
}

func dateComp[T ~int](d T) string {
	return strconv.FormatInt(int64(d), 10)
}

func dayPath(savePath string, date time.Time) string {
	return filepath.Join(filepath.Dir(savePath), dateComp(date.Year()), dateComp(date.Month()), dateComp(date.Day()))
}

// Save will add an entry to the root/tasks/{customerID}/{year}/{month}/{day}/{entryID}.json
func (e *Entry) Save(root string) error {
	entriesSavePath, err := e.Task.EnsureTaskEntriesFolder(root)
	if err != nil {
		return fmt.Errorf("could not create entries folder: %w", err)
	}
	entryPath := filepath.Join(dayPath(entriesSavePath, e.StartTS), e.ID.String()+".json")
	f, err := os.Create(entryPath)
	if err != nil {
		return fmt.Errorf("could not create entry file: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(e); err != nil {
		return fmt.Errorf("could not encode entry: %w", err)
	}
	return nil
}

// MarshalJSON method for Entry to be able to marshal task
func (e *Entry) MarshalJSON() ([]byte, error) {
	if e.Task == nil {
		return nil, errors.New("task is nil")
	}
	if e.Task.Customer == nil {
		return nil, errors.New("customer is nil")
	}
	// Create a shadow type to avoid infinite recursion
	alias := &struct {
		TaskID string `json:"task"`
		*Entry
	}{
		TaskID: path.Join(e.Task.Customer.ID.String(), e.Task.ID.String()),
		Entry:  e,
	}

	return json.MarshalIndent(alias, "", "  ")
}

// UnmarshalJSON method for Entry to be able to unmarshal task
func (e *Entry) UnmarshalJSON(data []byte) error {
	aux := &struct {
		TaskID string `json:"task"`
		*Entry
	}{
		Entry: e,
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	customerID, taskID := path.Split(aux.TaskID)
	customerID = strings.TrimSuffix(customerID, "/")
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Errorf("could not parse task ID: %w", err)
	}
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return fmt.Errorf("could not parse customer ID: %w", err)
	}
	c, ok := taskFromID[customerUUID]
	if !ok {
		return fmt.Errorf("could not find customer ID in task list %q", customerUUID)
	}
	task, ok := c[taskUUID]
	if !ok {
		return fmt.Errorf("could not find task %q", taskUUID)
	}
	e.Task = &task
	return nil
}

// Finish sets the end date for the given entry and persists result
func (e *Entry) Finish(root string) error {
	now := time.Now()
	if e.StartTS.YearDay() != now.YearDay() { //I am aware this breaks if you left it running for a year
		endTS := time.Date(e.StartTS.Year(), e.StartTS.Month(), e.StartTS.Day(), 23, 59, 59, 0, time.UTC)
		e.EndTs = &endTS
		// Now lets try to bridge the gap of  "various days of work"
		delta := e.StartTS.Sub(now)
		deltaInDays := int(delta.Hours() / 24)
		for i := 0; i < deltaInDays; i++ {
			var startT, endT time.Time
			startD := e.StartTS.Add(time.Duration(i+1) * 24 * time.Hour)
			startT = time.Date(startD.Year(), startD.Month(), startD.Day(), 0, 0, 0, 0, time.UTC)
			if i+1 == deltaInDays {
				endT = now
			} else {
				endT = time.Date(startD.Year(), startD.Month(), startD.Day(), 23, 59, 59, 0, time.UTC)
			}
			ee := &Entry{
				ID:      e.ID,
				Task:    e.Task,
				Comment: e.Comment,
				StartTS: startT,
				EndTs:   &endT,
			}
			if err := ee.Save(root); err != nil {
				return fmt.Errorf("could not save intermediate entry after finishing: %w", err)
			}
		}
	}
	e.EndTs = &now
	if err := e.Save(root); err != nil {
		return fmt.Errorf("could not save entry after finishing: %w", err)
	}
	return nil
}

type EntryStorage struct {
	Root           string
	VersionControl bool // git or not
	Online         bool // to push
}

// LoadCurrentEntry loads the latest open entry
func LoadCurrentEntry() (*Entry, error) {
	return &Entry{}, nil
}

// LoadEntryByID reads the given entry, if it exists
func LoadEntryByID(id uuid.UUID) (*Entry, error) {
	return &Entry{}, nil
}

// NewEntry instantiates an entry for the given task and customer.
func NewEntry(task *Task, now time.Time) *Entry {
	return &Entry{}
}

/*
Take in account loading entries for a task within a time frame, which is what report does
Also load entries for all tasks fo a customer within a time frame
Finally load all entries, for v2 for a time frame, useful for self reporting
*/
