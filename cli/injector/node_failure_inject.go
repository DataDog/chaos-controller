package main

import (
	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/injector"
	"github.com/spf13/cobra"
)

var nodeFailureInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a node failure on the actual node",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		shutdown, _ := cmd.Flags().GetBool("shutdown")

		i := injector.NodeFailureInjector{
			Injector: injector.Injector{
				UID: uid,
				Log: log,
			},
			Spec: &v1beta1.NodeFailureSpec{
				Shutdown: shutdown,
			},
		}
		i.Inject()
	},
}

func init() {
	nodeFailureInjectCmd.Flags().Bool("shutdown", false, "If specified, the host will shut down instead of reboot")
}
