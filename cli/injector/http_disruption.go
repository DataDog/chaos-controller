package main

import (
	"github.com/spf13/cobra"
)

var httpDisruptionCmd = &cobra.Command{
	Use:   "http-disruption",
	Short: "HTTP disruption subcommand",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		rawRequestFields, _ := cmd.Flags().GetStringArray("request-field")
		log.Infow("request fields", rawRequestFields)

		// for _, config := range configs {
		// 	inj, err := injector.NewHTTPDisruptionInjector(injector.HTTPDisruptionInjectorConfig{Config: config})
		// 	if err != nil {
		// 		log.Fatalw("error initializing the DNS injector", "error", err)
		// 	}

		// 	injectors = append(injectors, inj)
		// }
	},
}

func init() {
	// We must use a StringArray rather than StringSlice here, because our ip values can contain commas. StringSlice will split on commas.
	dnsDisruptionCmd.Flags().StringArray("host-record-pairs", []string{}, "list of host,record,value tuples as strings") // `foo.bar.svc.cluster.local;A;10.0.0.0,10.0.0.13`
	dnsDisruptionCmd.Flags().StringArray("http-port-list", []string{}, "list of host,record,value tuples as strings")    // `foo.bar.svc.cluster.local;A;10.0.0.0,10.0.0.13`
}
