package monitor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pilebones/go-udev/netlink"
	"github.com/primalmotion/battctl/internal/threshold"
	"github.com/primalmotion/battctl/internal/timerecord"
)

const (
	udevSubsystem            = "power_supply"
	udevEnvPowerSupplyOnline = "POWER_SUPPLY_ONLINE"
)

var matcher = &netlink.RuleDefinitions{
	Rules: []netlink.RuleDefinition{
		{
			Env: map[string]string{
				"SUBSYSTEM":              udevSubsystem,
				udevEnvPowerSupplyOnline: "1",
			},
		},
		{
			Env: map[string]string{
				"SUBSYSTEM":              udevSubsystem,
				udevEnvPowerSupplyOnline: "0",
			},
		},
	},
}

type Monitor struct {
	tr          *timerecord.TimeRecord
	dockedDelay time.Duration
	docked      threshold.Threshold
	mobileDelay time.Duration
	mobile      threshold.Threshold
}

func NewMonitor(tr *timerecord.TimeRecord, dockedDelay time.Duration, docked threshold.Threshold, mobileDelay time.Duration, mobile threshold.Threshold) *Monitor {

	return &Monitor{
		tr:          tr,
		dockedDelay: dockedDelay,
		docked:      docked,
		mobileDelay: mobileDelay,
		mobile:      mobile,
	}
}

func (m *Monitor) Run(ctx context.Context) error {

	// Connect to udev
	conn := &netlink.UEventConn{}
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		return fmt.Errorf("unable to connect to netlink kobject uevent socket: %w", err)
	}
	defer conn.Close()

	evts := make(chan netlink.UEvent)
	errs := make(chan error)
	quit := conn.Monitor(evts, errs, matcher)

	// Prepare timers
	timer := time.NewTimer(0)
	<-timer.C

	// Recover state
	if err := m.recover(timer); err != nil {
		return err
	}

	// Main loop
	for {

		select {

		case <-evts:

			wmode, delay, err := m.getWanted()
			if err != nil {
				return err
			}

			if wmode == m.tr.GetMode() {
				if err := m.tr.SetMode(wmode); err != nil {
					return err
				}
				timer.Stop()
				continue
			}

			if sremaining := m.tr.GetScheduleForMode(wmode); sremaining != 0 {
				delay = sremaining
			}

			if err := m.tr.SetScheduledMode(wmode, delay); err != nil {
				return err
			}

			timer.Reset(delay)

			fmt.Printf("scheduled: mode %s in %s\n", wmode, delay)

		case <-timer.C:

			wmode := m.tr.GetScheduledMode()

			var th threshold.Threshold
			if wmode == "docked" {
				th = m.docked
			} else {
				th = m.mobile
			}

			if err := m.tr.SetMode(wmode); err != nil {
				return err
			}

			if err := threshold.SetThreshold(th); err != nil {
				return err
			}

			fmt.Printf("enabled mode: %s (%s)\n", wmode, th)

		case err := <-errs:
			close(quit)
			return err

		case <-ctx.Done():
			close(quit)
			return ctx.Err()
		}
	}
}

func (m *Monitor) recover(timer *time.Timer) error {

	resetTimer := false
	remaining := time.Duration(0)

	mode := m.tr.GetMode()
	smode := m.tr.GetScheduledMode()
	wmode, _, err := m.getWanted()
	if err != nil {
		return err
	}

	if mode == "" {
		resetTimer = true
		mode = wmode
		if err := m.tr.SetScheduledMode(mode, 0); err != nil {
			return err
		}
		fmt.Println("restoring: mode initialized to", mode)
	}

	if wmode != mode && wmode != smode {
		resetTimer = true
		smode = "mobile"
		if err := m.tr.SetScheduledMode(smode, 0); err != nil {
			return err
		}
		fmt.Println("restoring: untracked changed. reinitialized to mobile")
	}

	if smode != "" {
		resetTimer = true
		remaining = m.tr.GetScheduleForMode(smode)
		fmt.Printf("restoring: scheduled mode %s in %s\n", smode, remaining)
	}

	if resetTimer {
		timer.Reset(remaining)
		fmt.Printf("restoring: firing restoration timer for %s in %s\n", mode, remaining)
	}

	fmt.Printf("restoring: state restoration complete: mode=%s smode=%s\n", mode, smode)
	return nil
}

func (m *Monitor) getWanted() (wmode string, delay time.Duration, err error) {

	online, err := isACOnline()
	if err != nil {
		return "", 0, err
	}

	if online {
		return "docked", m.dockedDelay, nil
	}

	return "mobile", m.mobileDelay, nil
}

func isACOnline() (bool, error) {

	data, err := os.ReadFile("/sys/class/power_supply/AC/online")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(data)) == "1", nil
}
