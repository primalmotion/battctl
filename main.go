package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/primalmotion/battctl/internal/monitor"
	"github.com/primalmotion/battctl/internal/state"
	"github.com/primalmotion/battctl/internal/threshold"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {

	cobra.OnInitialize(func() {
		viper.SetEnvPrefix("battctl")
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		viper.SetConfigName("conf")
		viper.AddConfigPath("/etc/battctl")
		viper.AddConfigPath("$home/.config/battclt")
		if err := viper.ReadInConfig(); err != nil {
			if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
				log.Fatal("unable to read config: ", err)
			}
		}
	})

	var rootCmd = &cobra.Command{
		Use:          "batthctl",
		SilenceUsage: true,
	}

	var cmdGet = &cobra.Command{
		Use:   "get",
		Short: "Get the current charge thresholds.",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			th, err := threshold.GetThreshold()
			if err != nil {
				return err
			}
			fmt.Println(th)
			return nil
		},
	}

	var cmdSet = &cobra.Command{
		Use:   "set <start:int> <end:int>",
		Short: "Sets the start and end threshold values",
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

			return threshold.SetThreshold(threshold.Threshold{Start: start, End: end})
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

			fmt.Printf("conf: data-dir=%s data-clean=%t\n", dataDir, dataClean)
			fmt.Printf("conf: docked: delay=%s start=%d end=%d\n", dockedDelay, dockedStart, dockedEnd)
			fmt.Printf("conf: mobile: delay=%s start=%d end=%d\n", mobileDelay, mobileStart, mobileEnd)

			st := state.New(path.Join(dataDir, "state"))
			if err := st.Load(); err != nil {
				return err
			}

			return monitor.NewMonitor(
				st,
				dockedDelay,
				threshold.Threshold{
					Start: dockedStart,
					End:   dockedEnd,
				},
				mobileDelay,
				threshold.Threshold{
					Start: mobileStart,
					End:   mobileEnd,
				},
			).Run(context.Background())
		},
	}
	cmdMonitor.Flags().DurationP("docked-delay", "d", 24*time.Hour, "How long to wait before setting docked mode after power supply is plugged")
	cmdMonitor.Flags().IntP("docked-start", "s", 40, "Value for charge control threshold start in docked mode")
	cmdMonitor.Flags().IntP("docked-end", "e", 95, "Value for charge control threshold end in docked mode")
	cmdMonitor.Flags().DurationP("mobile-delay", "D", 1*time.Minute, "How long to wait before setting mobile mode after power supply is unplugged")
	cmdMonitor.Flags().IntP("mobile-start", "S", 90, "Value for charge control threshold start on battery")
	cmdMonitor.Flags().IntP("mobile-end", "E", 95, "Value for charge control threshold end on mobile")
	cmdMonitor.Flags().String("data-dir", "/var/lib/battctl", "Path to data folder.")
	cmdMonitor.Flags().Bool("data-clean", false, "Delete content of data folder before starting.")

	rootCmd.AddCommand(
		cmdGet,
		cmdSet,
		cmdMonitor,
	)

	rootCmd.Execute()
}
