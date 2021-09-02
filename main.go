package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {

	cobra.OnInitialize(func() {
		viper.SetEnvPrefix("battctl")
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	})

	var rootCmd = &cobra.Command{
		Use: "batthctl",
	}

	var cmdGet = &cobra.Command{
		Use:   "get",
		Short: "get the current charge thresholds.",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			th, err := GetThreshold()
			if err != nil {
				return err
			}
			fmt.Println(th)
			return nil
		},
	}

	var cmdSet = &cobra.Command{
		Use:   "set",
		Short: "Sets the threshold values",
		Args:  cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			end, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			return SetThreshold(Threshold{Start: start, endEnd: end})
		},
	}

	var cmdMonitor = &cobra.Command{
		Use:   "monitor",
		Short: "Starts the monitor daemon",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			dockedDelay := viper.GetDuration("docked-delay")
			dockedStart := viper.GetInt("docked-start")
			dockedEnd := viper.GetInt("docked-end")
			mobileDelay := viper.GetDuration("mobile-delay")
			mobileStart := viper.GetInt("mobile-start")
			mobileEnd := viper.GetInt("mobile-end")
			dataDir := viper.GetString("data-dir")
			dataClean := viper.GetBool("data-clean")

			if dataClean {
				if err := os.RemoveAll(dataDir); err != nil {
					return err
				}
			}
			if err := os.MkdirAll(dataDir, 0700); err != nil {
				return err
			}

			fmt.Println("starting monitor")
			fmt.Printf("docked: delay=%s start=%d end=%d\n", dockedDelay, dockedStart, dockedEnd)
			fmt.Printf("mobile: delay=%s start=%d end=%d\n", mobileDelay, mobileStart, mobileEnd)

			return NewMonitor(
				NewTimeRecorder(path.Join(dataDir, "battdata")),
				dockedDelay,
				Threshold{
					Start:  dockedStart,
					endEnd: dockedEnd,
				},
				mobileDelay,
				Threshold{
					Start:  mobileStart,
					endEnd: mobileEnd,
				},
			).Run(context.Background())
		},
	}
	cmdMonitor.Flags().DurationP("docked-delay", "d", 24*time.Hour, "how long to wait to set docked mode after power supply is plugged")
	cmdMonitor.Flags().IntP("docked-start", "s", 90, "value for charge control threshold start on AC")
	cmdMonitor.Flags().IntP("docked-end", "e", 95, "value for charge control threshold end on AC")
	cmdMonitor.Flags().DurationP("mobile-delay", "D", 1*time.Minute, "how long to wait to set mobile mode after power supply is unplugged")
	cmdMonitor.Flags().IntP("mobile-start", "S", 91, "value for charge control threshold start on battery")
	cmdMonitor.Flags().IntP("mobile-end", "E", 96, "value for charge control threshold end on battery")
	cmdMonitor.Flags().String("data-dir", "/var/lib/battctl", "path to a file to kee track of timings")
	cmdMonitor.Flags().Bool("data-clean", false, "delete time record file before starting")

	rootCmd.AddCommand(
		cmdGet,
		cmdSet,
		cmdMonitor,
	)

	rootCmd.Execute()
}
