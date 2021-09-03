package timerecord

import (
	"encoding/json"
	"os"
	"time"
)

type TimeRecord struct {
	Mode          string    `json:"mode"`
	ScheduledMode string    `json:"scheduledMode"`
	ScheduledTime time.Time `json:"scheduledTime"`

	path string
}

func New(path string) *TimeRecord {
	return &TimeRecord{
		path: path,
	}
}

func (t *TimeRecord) Load() error {
	data, err := os.ReadFile(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, t)
}

func (t *TimeRecord) Save() error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.path, data, 0600)
}

func (t *TimeRecord) GetMode() string {
	return t.Mode
}

func (t *TimeRecord) SetMode(mode string) error {

	t.Mode = mode

	t.ScheduledMode = ""
	t.ScheduledTime = time.Time{}

	return t.Save()
}

func (t *TimeRecord) GetScheduledMode() (mode string) {
	return t.ScheduledMode
}

func (t *TimeRecord) SetScheduledMode(mode string, in time.Duration) error {

	t.ScheduledMode = mode
	t.ScheduledTime = time.Now().Add(in)

	return t.Save()
}

func (t *TimeRecord) GetScheduleForMode(mode string) (remaining time.Duration) {

	remaining = time.Until(t.ScheduledTime)

	if remaining < 0 {
		remaining = 0
	}

	return remaining
}
