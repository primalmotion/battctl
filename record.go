package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"time"
)

const TimeRecordModeUnset = "-"

type TimeRecord struct {
	path string
	mode string
}

func NewTimeRecorder(path string) *TimeRecord {
	return &TimeRecord{
		path: path,
		mode: TimeRecordModeUnset,
	}
}

func (t *TimeRecord) Record(mode string) error {

	t.mode = mode

	return os.WriteFile(
		t.path,
		[]byte(fmt.Sprintf("%s:%d", mode, time.Now().Unix())),
		0600,
	)
}

func (t *TimeRecord) GetMode() string {
	return t.mode
}

func (t *TimeRecord) IsMode(mode string) bool {
	return t.mode == mode
}

func (t *TimeRecord) Delete() error {
	err := os.Remove(t.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (t *TimeRecord) Since() (time.Duration, error) {

	data, err := os.ReadFile(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	parts := bytes.SplitN(data, []byte{':'}, 2)

	d, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		return 0, fmt.Errorf("unable to convert recorded time: %w", err)
	}

	t.mode = string(parts[0])

	return time.Since(time.Unix(int64(d), 0)), nil
}
