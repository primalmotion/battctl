# battctl

battctl is a tool that allows you to:

- get the current charge threshold values
- set the charge threshold values
- monitor AC status through udev to apply one the the 2 charging modes

The thresholds for both modes (`mobile` or `docked`) can be individually
configured. Each mode comes with a delay to define how long to wait
after the AC event to switch to the mode.

For instance, if the `docked` delay is 24h and the `mobile` delay is 30m,
this means that 24h hours after plugging the AC in, the thresholds will be
updated to the `docked` values. If you unplug the AC for less than 30m,
the thresholds will not change. After 30 minutes, the thresholds will be
restored to the `mobile` values. If you plug the AC back, it will take
24h to switch to the `docked` values again.

> This requires your battery thresholds to be exposed in user-space.
> This program assumes the following paths exist by default:
>
> - `/sys/class/power_supply/BAT0/charge_control_end_threshold`
> - `/sys/class/power_supply/BAT0/charge_control_end_threshold`
>
> They can be changed using flags.
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
	  battctl monitor [flags]

	Flags:
	      --data-clean                    Delete content of data folder before starting.
	      --data-dir string               Path to data folder. (default "/var/lib/battctl")
	  -d, --docked-delay duration         How long to wait before setting docked mode after power supply is plugged (default 24h0m0s)
	  -e, --docked-end int                Value for charge control threshold end in docked mode (default 95)
	  -s, --docked-start int              Value for charge control threshold start in docked mode (default 40)
	  -h, --help                          help for monitor
	  -D, --mobile-delay duration         How long to wait before setting mobile mode after power supply is unplugged (default 1m0s)
	  -E, --mobile-end int                Value for charge control threshold end on mobile (default 95)
	  -S, --mobile-start int              Value for charge control threshold start on battery (default 90)
	      --threshold-end-path string     Path to the charge control end file (default "/sys/class/power_supply/BAT0/charge_control_end_threshold")
	      --threshold-start-path string   Path to the charge control start file (default "/sys/class/power_supply/BAT0/charge_control_start_threshold")

### systemd unit

A systemd unit file can be found in the `dist` folder to deal with the monitor.

### Config file

These values can be configured by creating a file in `/etc/battctl/conf.yaml`
where the keys are the flags without their `--` prefix.

For example:

	docked-delay: 48h
	docked-start: 50
	docked-end: 90

An example of a config file can be found in the `dist` folder.

### State

The monitor keep tracks of the last mode and last time of even in
`/var/lib/battctl/state` by default.
