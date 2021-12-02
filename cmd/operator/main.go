package main

import (
	"eventrigger.com/operator/pkg/manager"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	opt := &manager.OperatorOptions{}

	rootCmd := &cobra.Command{
		Use:   "operator",
		Short: "Event trigger operator",
		RunE: func(cmd *cobra.Command, args []string) error {

			operator, err := manager.NewOperator(opt)
			if err != nil {
				return err
			}
			return operator.Run()
		},
	}
	rootCmd.Flags().IntVar(&opt.Port, "port", 8081, "Http Server Port")
	rootCmd.Flags().UintVar(&opt.CloudEventsPort, "cloud-events-port", 7787, "Cloud Events Port")
	rootCmd.Flags().IntVar(&opt.MetricsPort, "metrics-port", 7788, "Operator Metrics Port")
	rootCmd.Flags().IntVar(&opt.HealthPort, "health-port", 7789, "Operator Health Port")
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("exit with err: %s \n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
