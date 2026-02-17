package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	outputFmt  string
	levelFilter string
)

// rootCmd is the base command when called without subcommands.
var rootCmd = &cobra.Command{
	Use:   "loom",
	Short: "Loom — Log-Observer & Monitor",
	Long: `Loom is a high-performance, real-time log aggregation CLI tool.
It monitors multiple log files, extracts structured data via pattern matching,
and provides instant observability through your terminal and a live web dashboard.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: $HOME/.loom.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "text", "output format: text, json")
	rootCmd.PersistentFlags().StringVarP(&levelFilter, "level", "l", "", "filter by severity (comma-separated: info,warn,error)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".loom")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()
	_ = viper.ReadInConfig()
}
