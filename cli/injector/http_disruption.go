package main

import (
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var httpDisruptionCmd = &cobra.Command{
	Use:   "http-disruption",
	Short: "HTTP disruption subcommand",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		rawRequestFields, _ := cmd.Flags().GetStringArray("request-field")
		spec := []v1beta1.TargetDomain{}

		for _, rawField := range rawRequestFields {
			fields := strings.Split(rawField, ";")
			if len(fields) != 3 {
				log.Fatalw("could not parse --host-record-pairs argument to dns-disruption", "offending argument", rawField)
				continue
			}

			spec = append(spec, v1beta1.TargetDomain{
				Domain: fields[0],
				Header: []v1beta1.RequestField{
					{
						Uri:    fields[1],
						Method: fields[2],
					},
				},
			})
		}

		for _, config := range configs {
			inj, err := injector.NewHTTPDisruptionInjector(spec, injector.HTTPDisruptionInjectorConfig{Config: config})
			if err != nil {
				log.Fatalw("error initializing the DNS injector", "error", err)
			}

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	// We must use a StringArray rather than StringSlice here, because our ip values can contain commas. StringSlice will split on commas.
	httpDisruptionCmd.Flags().StringSlice("http-port-list", []string{}, "list of comma-delineated port values for http traffic as strings")   // `80,8080`
	httpDisruptionCmd.Flags().StringSlice("https-port-list", []string{}, "list of comma-delineated port values for https traffic as strings") // `443,8443`
	httpDisruptionCmd.Flags().StringSlice("request-field", []string{}, "list of domain, uri, method values as strings")                       // `foo.com/bar/baz;GET
}
