// Package cmd contains command execution logic.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "taskmaster",
	Short: "Aggregate activity data and generate AI-powered standups, summaries, and suggestions",
	Long: `Taskmaster aggregates your activity data from Slack, Jira, Todoist, and Notion,
caches it locally, and uses Gemini to generate daily standups, weekly summaries,
and prioritization suggestions.`,
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetConfigName(".taskmaster")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("TASKMASTER")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config: %s\n", err)
		}
	}
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
