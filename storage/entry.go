package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path"
	"path/filepath"
	"sort"
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

// Entries is a slice of Entry, it implements the required methods for sorting
type Entries []*Entry

// Len returns the length of the slice, it is required for sorting
func (e Entries) Len() int {
	return len(e)
}

// Less compares two entries by their start date, it is required for sorting
func (e Entries) Less(i, j int) bool {
	return e[i].StartTS.Before(e[j].StartTS)
}

// Swap swaps two entries, it is required for sorting
func (e Entries) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func dateComp[T ~int](d T) string {
	return strconv.FormatInt(int64(d), 10)
}

func dayPath(savePath string, date time.Time) string {
	return filepath.Join(filepath.Dir(savePath), dateComp(date.Year()), dateComp(date.Month()), dateComp(date.Day()))
}

// latestDateFolder returns the latest date folder in the given root
func latestDateFolder(root string) (string, error) {
	years, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("reading root directory: %w", err)
	}

	var yearDirs []int
	for _, year := range years {
		if year.IsDir() {
			yearInt, err := strconv.Atoi(year.Name())
			if err == nil {
				yearDirs = append(yearDirs, yearInt)
			}
		}
	}

	if len(yearDirs) == 0 {
		return "", fmt.Errorf("no valid year directories found")
	}

	sort.Sort(sort.Reverse(sort.IntSlice(yearDirs)))
	latestYear := yearDirs[0]
	yearPath := filepath.Join(root, strconv.Itoa(latestYear))

	months, err := os.ReadDir(yearPath)
	if err != nil {
		return "", fmt.Errorf("reading year directory: %w", err)
	}

	var monthDirs = make([]int, 0, 12)
	for _, month := range months {
		if month.IsDir() {
			monthInt, err := strconv.Atoi(month.Name())
			if err == nil {
				monthDirs = append(monthDirs, monthInt)
			}
		}
	}

	if len(monthDirs) == 0 {
		return "", fmt.Errorf("no valid month directories found")
	}

	sort.Sort(sort.Reverse(sort.IntSlice(monthDirs)))
	latestMonth := monthDirs[0]
	monthPath := filepath.Join(yearPath, strconv.Itoa(latestMonth))

	days, err := os.ReadDir(monthPath)
	if err != nil {
		return "", fmt.Errorf("reading month directory: %w", err)
	}

	var dayDirs = make([]int, 0, 31)
	for _, day := range days {
		if day.IsDir() {
			dayInt, err := strconv.Atoi(day.Name())
			if err == nil {
				dayDirs = append(dayDirs, dayInt)
			}
		}
	}

	if len(dayDirs) == 0 {
		return "", fmt.Errorf("no valid day directories found")
	}

	sort.Sort(sort.Reverse(sort.IntSlice(dayDirs)))
	latestDay := dayDirs[0]
	latestDayPath := filepath.Join(monthPath, strconv.Itoa(latestDay))

	return latestDayPath, nil
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

// LoadLatestDayEntries loads all entries for the given customer on the latest date available
func LoadLatestDayEntries(root string, customer *Customer) ([]*Entry, error) {
	latestDayPath, err := latestDateFolder(EntriesSavePath(root, customer))
	if err != nil {
		return nil, fmt.Errorf("could not find latest date folder: %w", err)
	}
	return LoadPathEntries(latestDayPath)
}

// LoadDayEntries loads all entries for the given customer on the given date
func LoadDayEntries(root string, customer *Customer, date time.Time) ([]*Entry, error) {
	esp := EntriesSavePath(root, customer)
	dayEntriesPath := dayPath(esp, date)
	return LoadPathEntries(dayEntriesPath)
}

func LoadPathEntries(entriesPath string) ([]*Entry, error) {
	// now load all entries for that day which are in the form of json files
	entries, err := os.ReadDir(entriesPath)
	if err != nil {
		return nil, fmt.Errorf("reading day directory: %w", err)
	}
	var dayEntries = make([]*Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryPath := filepath.Join(entriesPath, entry.Name())
		f, err := os.Open(entryPath)
		if err != nil {
			return nil, fmt.Errorf("open entry file: %w", err)
		}
		var e Entry
		err = json.NewDecoder(f).Decode(&e)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("decode entry file: %w", err)
		}
		dayEntries = append(dayEntries, &e)
	}
	sort.Sort(Entries(dayEntries))
	return dayEntries, nil
}

// LoadCurrentEntry loads the latest open entry for the given customer.
func LoadCurrentEntry(root string, customer *Customer) (*Entry, error) {
	dayEntries, err := LoadDayEntries(root, customer, time.Now())
	if err != nil {
		return nil, fmt.Errorf("loading day entries: %w", err)
	}
	if len(dayEntries) == 0 {
		return nil, ErrNotFound
	}
	return dayEntries[len(dayEntries)-1], nil
}

// NewEntry instantiates an entry for the given task and customer.
func NewEntry(task *Task, now time.Time) *Entry {
	return &Entry{
		ID:      uuid.New(),
		Task:    task,
		StartTS: now,
	}
}

/*
Take in account loading entries for a task within a time frame, which is what report does
Also load entries for all tasks fo a customer within a time frame
Finally load all entries, for v2 for a time frame, useful for self reporting
*/
