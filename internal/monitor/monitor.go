package monitor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pilebones/go-udev/netlink"
	"github.com/primalmotion/battctl/internal/state"
	"github.com/primalmotion/battctl/internal/threshold"
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
	st          *state.State
	dockedDelay time.Duration
	docked      threshold.Threshold
	mobileDelay time.Duration
	mobile      threshold.Threshold
}

func NewMonitor(tr *state.State, dockedDelay time.Duration, docked threshold.Threshold, mobileDelay time.Duration, mobile threshold.Threshold) *Monitor {

	return &Monitor{
		st:          tr,
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

	evts := make(chan netlink.UEvent, 2)
	errs := make(chan error)
	quit := conn.Monitor(evts, errs, matcher)

	// Prepare timers
	timer := time.NewTimer(0)
	<-timer.C
	ticker := time.NewTicker(10 * time.Minute)

	// Recover state
	if err := m.sync(timer); err != nil {
		return err
	}

	// Main loop
	for {

		select {

		case <-ticker.C:
			if err := m.sync(timer); err != nil {
				return err
			}

		case <-evts:

			wmode, wdelay, err := m.getWanted()
			if err != nil {
				return err
			}

			if err := m.st.Load(); err != nil {
				return err
			}

			if wmode == m.st.GetMode() {
				if err := m.st.SetMode(wmode); err != nil {
					return err
				}
				timer.Stop()
				continue
			}

			if sremaining := m.st.GetScheduleForMode(wmode); sremaining != 0 {
				wdelay = sremaining
			}

			if err := m.st.SetScheduledMode(wmode, wdelay); err != nil {
				return err
			}

			timer.Reset(wdelay)

			fmt.Printf("scheduled: mode %s in %s\n", wmode, wdelay)

		case <-timer.C:

			if err := m.st.Load(); err != nil {
				return err
			}

			wmode := m.st.GetScheduledMode()

			var th threshold.Threshold
			if wmode == "docked" {
				th = m.docked
			} else {
				th = m.mobile
			}

			if err := m.st.SetMode(wmode); err != nil {
				return err
			}

			if err := threshold.SetThreshold(th); err != nil {
				return err
			}

			fmt.Printf("mode: %s (%s)\n", wmode, th)

		case err := <-errs:
			close(quit)
			return err

		case <-ctx.Done():
			close(quit)
			return ctx.Err()
		}
	}
}

func (m *Monitor) sync(timer *time.Timer) error {

	if err := m.st.Load(); err != nil {
		return err
	}

	resetTimer := false
	remaining := time.Duration(0)

	mode := m.st.GetMode()
	smode := m.st.GetScheduledMode()
	wmode, _, err := m.getWanted()
	if err != nil {
		return err
	}

	if mode == "" {
		resetTimer = true
		mode = wmode
		if err := m.st.SetScheduledMode(mode, 0); err != nil {
			return err
		}
		fmt.Println("sync: no state. mode initialized to", mode)
	}

	if wmode != mode && wmode != smode {
		resetTimer = true
		smode = wmode
		if err := m.st.SetScheduledMode(smode, 0); err != nil {
			return err
		}
		fmt.Println("sync: untracked changed. reinitialized to", wmode)
	}

	if smode != "" {
		resetTimer = true
		remaining = m.st.GetScheduleForMode(smode)
		fmt.Printf("sync: scheduled mode %s in %s\n", smode, remaining)
	}

	if resetTimer {
		timer.Reset(remaining)
	}

	fmt.Printf("sync: state sync complete: mode=%s smode=%s\n", mode, smode)

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
