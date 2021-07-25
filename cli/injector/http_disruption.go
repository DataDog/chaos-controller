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
	httpDisruptionCmd.Flags().StringArray("http-port-list", []string{}, "list of comma-delineated port values for http traffic as strings")   // `80,8080`
	httpDisruptionCmd.Flags().StringArray("https-port-list", []string{}, "list of comma-delineated port values for https traffic as strings") // `443,8443`
	httpDisruptionCmd.Flags().StringArray("request-field", []string{}, "list of domain, uri, method values as strings")                       // `foo.com/bar/baz;GET
}
