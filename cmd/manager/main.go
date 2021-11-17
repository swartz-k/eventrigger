package main

import (
	"eventrigger.com/operator/pkg/controllers"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	opt := &controllers.ManagerOptions{}

	rootCmd := &cobra.Command{
		Use:   "manager",
		Short: "Event trigger manager",
		RunE: func(cmd *cobra.Command, args []string) error {

			manager := controllers.NewManager(opt)
			return manager.Run()
		},
	}
	rootCmd.Flags().UintVar(&opt.CloudEventsPort, "cloud-events-port", 7789, "Cloud Events Port")

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("exit with err: %s \n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
