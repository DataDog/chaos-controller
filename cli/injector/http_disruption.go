package main

import (
	"strconv"
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
		httpPortVals := []int{}
		httpsPortVals := []int{}

		// Handle http port args
		rawRequestFields, _ := cmd.Flags().GetStringArray("http-port-list")
		log.Info("arguments to httpDisruptionCmd", "http-port-list", rawRequestFields)
		if len(rawRequestFields) != 0 {
			for _, field := range rawRequestFields {
				port, err := strconv.Atoi(field)
				if err != nil {
					log.Fatalw("failed to parse port value, skipping:", field)
					continue
				}

				httpPortVals = append(httpPortVals, port)
			}
		} else {
			log.Info("using default http port: 80")
			httpPortVals = append(httpPortVals, 80)
		}

		// Handle https port args
		rawRequestFields, _ = cmd.Flags().GetStringArray("https-port-list")
		log.Info("arguments to httpDisruptionCmd", "https-port-list", rawRequestFields)
		if len(rawRequestFields) != 0 {
			for _, field := range rawRequestFields {
				port, err := strconv.Atoi(field)
				if err != nil {
					log.Fatalw("failed to parse port value, skipping:", field)
					continue
				}

				httpsPortVals = append(httpsPortVals, port)
			}
		} else {
			log.Info("using default https port: 443")
			httpsPortVals = append(httpsPortVals, 443)
		}

		spec := v1beta1.HTTPDisruptionSpec{
			Domains:    []v1beta1.TargetDomain{},
			HttpPorts:  httpPortVals,
			HttpsPorts: httpsPortVals,
		}

		// Handle domain/uri/http method tuples
		rawRequestFields, _ = cmd.Flags().GetStringArray("request-field")
		log.Info("arguments to httpDisruptionCmd", "request-field", rawRequestFields)
		for _, rawField := range rawRequestFields {
			fields := strings.Split(rawField, ";")
			if len(fields) != 3 {
				log.Fatalw("could not parse --request-field argument to http-disruption", "offending argument", rawField)
				continue
			}

			spec.Domains = append(spec.Domains, v1beta1.TargetDomain{
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
				log.Fatalw("error initializing the HTTP injector", "error", err)
			}

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	httpDisruptionCmd.Flags().StringArray("http-port-list", []string{}, "list of port values for http traffic as strings")   // `80`
	httpDisruptionCmd.Flags().StringArray("https-port-list", []string{}, "list of port values for https traffic as strings") // `443`
	httpDisruptionCmd.Flags().StringArray("request-field", []string{}, "list of domain, uri, method values as strings")      // `foo.com;/bar/baz;GET
}
