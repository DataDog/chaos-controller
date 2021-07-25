package main

import (
	//"github.com/DataDog/chaos-controller/api/v1beta1"
	//"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var httpDisruptionCmd = &cobra.Command{
	Use:   "http-disruption",
	Short: "HTTP disruption subcommand",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		rawRequestFields, _ := cmd.Flags().GetStringArray("request-field")
		log.Infow("request fields", rawRequestFields)
		//for _, config := range configs {
		//	inj, err := injector.NewHTTPDisruptionInjector(injector.HTTPDisruptionInjectorConfig{Config: config})
		//	if err != nil {
		//		log.Fatalw("error initializing the DNS injector", "error", err)
		//	}

		//	injectors = append(injectors, inj)
		//}
	},
}
