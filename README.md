# battctl

battctl is a tool that allows:

- getting the current charge threshold values
- setting the current charge threshold values
- monitor AC status through udev to apply one the the 2 charging profiles

The thresholds for both profiles (mobile or docked) can be individually 
configured. Each profile comes with a delay, that allows to define how 
long to wait before switching after the AC event.

For instance, if the docked delay is 24h and the mobile delay is 30m, this
means that 24h hours after docking, the thresholds will be updated to their
docked values. If you unplug the AC for less than 30m, the thresholds will 
stay at their docked values. After 30 minutes, the thresholds will be 
restored to their mobile value. If you plug the AC back, it will take 
24h again to switch to the docked values.

> This require your battery thresolds to be exposed in user-space.
> This progam assumes you have following paths accessible:
> 
> - /sys/class/power_supply/BAT0/charge_control_end_threshold
> - /sys/class/power_supply/BAT0/charge_control_end_threshold
> 
> This has only be tested on a Purism Librem 14. For anything else
> patches welcome!

## Getting / Setting

To get the current values:

	battctl get

To set the values:

	battctl set 90 95

Where the first argument is the charge threshold start and the second the
charge threshold end.

## Monitoring

To start the monitor:

	battctl monitor

### Flags

There are several options that you can use to set for how long to
wait to switch from one mode to the other, and what thresholds to apply.

	battctl monitor -h
	Starts the monitor daemon

	Usage:
	  batthctl monitor [flags]

	Flags:
	      --data-clean              Delete content of data folder before starting.
	      --data-dir string         Path to data folder. (default "/var/lib/battctl")
	  -d, --docked-delay duration   How long to wait before setting docked mode after power supply is plugged (default 24h0m0s)
	  -e, --docked-end int          Value for charge control threshold end in docked mode (default 95)
	  -s, --docked-start int        Value for charge control threshold start in docked mode (default 40)
	  -h, --help                    help for monitor
	  -D, --mobile-delay duration   How long to wait before setting mobile mode after power supply is unplugged (default 1m0s)
	  -E, --mobile-end int          Value for charge control threshold end on mobile (default 95)
	  -S, --mobile-start int        Value for charge control threshold start on battery (default 90)

### Config file

These values can be configured by creating a file in `/etc/battctl/conf.yaml`
where the keys are the flags without their `--` prefix. For example:

	docked-delay: 48h
	docked-start: 50
	docked-end: 90

### State

The monitor keep tracks of the last mode and last time of even in `/var/lib/battctl/battdata` by default.
