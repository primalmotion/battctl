package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pilebones/go-udev/netlink"
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
	tr          *TimeRecord
	dockedDelay time.Duration
	docked      Threshold
	mobileDelay time.Duration
	mobile      Threshold
}

func NewMonitor(tr *TimeRecord, dockedDelay time.Duration, docked Threshold, mobileDelay time.Duration, mobile Threshold) *Monitor {

	return &Monitor{
		tr:          tr,
		dockedDelay: dockedDelay,
		docked:      docked,
		mobileDelay: mobileDelay,
		mobile:      mobile,
	}
}

func (m *Monitor) Run(ctx context.Context) error {

	// Connect to udev`
	conn := &netlink.UEventConn{}
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		return fmt.Errorf("unable to connect to netlink kobject uevent socket: %w", err)
	}
	defer conn.Close()

	evts := make(chan netlink.UEvent)
	errs := make(chan error)
	quit := conn.Monitor(evts, errs, matcher)

	// Prepare timers
	dockedTimer := time.NewTimer(0)
	<-dockedTimer.C
	mobileTimer := time.NewTimer(0)
	<-mobileTimer.C

	// Verify current state
	acOnline, err := isACOnline()
	if err != nil {
		return err
	}

	sr, err := m.tr.Since()
	if err != nil {
		return fmt.Errorf("restoring: unable to get record.since: %w", err)
	}

	switch m.tr.GetMode() {

	case "mobile":
		if acOnline {
			dockedTimer.Reset(m.dockedDelay)
			fmt.Println("restoring: ac=online mode=mobile: untracked switch. scheduling docked in", m.dockedDelay)
		} else {
			remaining := m.mobileDelay - sr
			if remaining < 0 {
				remaining = 0
			}
			mobileTimer.Reset(remaining)
			fmt.Println("restoring: ac=offline mode=mobile: scheduling mobile in", remaining)
		}

	case "docked":
		if !acOnline {
			mobileTimer.Reset(m.mobileDelay)
			fmt.Println("restoring: ac=offline mode=docked: untracked switch. scheduling mobile in", m.mobileDelay)
		} else {
			remaining := m.dockedDelay - sr
			if remaining < 0 {
				remaining = 0
			}
			dockedTimer.Reset(remaining)
			fmt.Println("restoring: ac=online mode=docked: scheduling docked in", remaining)
		}

	case TimeRecordModeUnset:
		if acOnline {
			dockedTimer.Reset(0)
			m.tr.Record("docked")
			fmt.Println("restoring: unset state, setting docked now")
		} else {
			mobileTimer.Reset(0)
			m.tr.Record("mobile")
			fmt.Println("restoring: unset state, setting mobile now")
		}
	}

	// Main loop
	for {

		select {

		case evt := <-evts:
			switch evt.Env[udevEnvPowerSupplyOnline] {
			case "0":
				if !m.tr.IsMode("mobile") {
					dockedTimer.Stop()
					mobileTimer.Reset(m.mobileDelay)
					m.tr.Record("mobile")
					fmt.Println("scheduled: mode mobile in", m.mobileDelay)
				}

			case "1":
				if !m.tr.IsMode("docked") {
					mobileTimer.Stop()
					dockedTimer.Reset(m.dockedDelay)
					m.tr.Record("docked")
					fmt.Println("scheduled: mode docked in", m.dockedDelay)
				}
			}

		case <-dockedTimer.C:
			if err := SetThreshold(m.docked); err != nil {
				return err
			}
			fmt.Printf("enabled mode: docked (%s)\n", m.docked)

		case <-mobileTimer.C:
			if err := SetThreshold(m.mobile); err != nil {
				return err
			}
			fmt.Printf("enabled mode: mobile (%s)\n", m.mobile)

		case err := <-errs:
			close(quit)
			return err

		case <-ctx.Done():
			close(quit)
			return ctx.Err()
		}
	}
}

func isACOnline() (bool, error) {

	data, err := os.ReadFile("/sys/class/power_supply/AC/online")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(data)) == "1", nil
}
