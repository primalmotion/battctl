package state

import (
	"encoding/json"
	"os"
	"time"
)

type State struct {
	s struct {
		Mode          string    `json:"mode"`
		ScheduledMode string    `json:"scheduledMode"`
		ScheduledTime time.Time `json:"scheduledTime"`
	}
	path string
}

func New(path string) *State {
	return &State{
		path: path,
	}
}

func (t *State) Load() error {
	data, err := os.ReadFile(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &t.s)
}

func (t *State) Save() error {
	data, err := json.MarshalIndent(t.s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.path, data, 0600)
}

func (t *State) GetMode() string {
	return t.s.Mode
}

func (t *State) SetMode(mode string) error {

	t.s.Mode = mode

	t.s.ScheduledMode = ""
	t.s.ScheduledTime = time.Time{}

	return t.Save()
}

func (t *State) GetScheduledMode() (mode string) {
	return t.s.ScheduledMode
}

func (t *State) SetScheduledMode(mode string, in time.Duration) error {

	t.s.ScheduledMode = mode
	t.s.ScheduledTime = time.Now().Add(in)

	return t.Save()
}

func (t *State) GetScheduleForMode(mode string) (remaining time.Duration) {

	remaining = time.Until(t.s.ScheduledTime)

	if remaining < 0 {
		remaining = 0
	}

	return remaining
}
