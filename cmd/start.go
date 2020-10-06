package cmd

import (
	backend "github.com/anvh2/notification-server/service"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Notification Server",
	Long:  `Start Notification Server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := backend.NewServer()
		return server.Run()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
