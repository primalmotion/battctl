package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	chargeControlStartThresoldPath = "/sys/class/power_supply/BAT0/charge_control_start_threshold"
	chargeControlEndThresoldPath   = "/sys/class/power_supply/BAT0/charge_control_end_threshold"
)

type Threshold struct {
	Start  int
	endEnd int
}

func (t Threshold) String() string {
	return fmt.Sprintf("start:%d end:%d", t.Start, t.endEnd)
}

func GetThreshold() (Threshold, error) {

	var (
		out Threshold
		err error
	)

	out.Start, err = readThreshold(chargeControlStartThresoldPath)
	if err != nil {
		return out, fmt.Errorf("unable to read start threshold file: %w", err)
	}

	out.endEnd, err = readThreshold(chargeControlEndThresoldPath)
	if err != nil {
		return out, fmt.Errorf("unable to write end file: %w", err)
	}

	return out, nil
}

func SetThreshold(th Threshold) error {

	if err := writeThreshold(chargeControlStartThresoldPath, th.Start); err != nil {
		return fmt.Errorf("unable to write start file: %w", err)
	}

	if err := writeThreshold(chargeControlEndThresoldPath, th.endEnd); err != nil {
		return fmt.Errorf("unable to write end file: %w", err)
	}

	return nil
}

func writeThreshold(path string, value int) error {

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 644)
	if err != nil {
		return fmt.Errorf("unable to set open '%s': %w", path, err)
	}

	if _, err := f.Write([]byte(strconv.Itoa(value))); err != nil {
		return fmt.Errorf("unable to write '%s': %w", path, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("unable to close '%s': %w", path, err)
	}

	return nil
}

func readThreshold(path string) (int, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("unable to read '%s': %w", path, err)
	}

	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("unable to convert '%s': %w", string(data), err)
	}

	return v, nil
}
