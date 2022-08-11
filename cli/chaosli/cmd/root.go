// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/ddmark"
	"github.com/DataDog/chaos-controller/types"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// Version will be set with the -ldflags option at compile time
var Version = "v0"
var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "chaosli",
	Short: "Chaos Controller CLI to aid with Disruption Configurations.",
	Long: `
Chaos Controller CLI that will aid with dealing with Disruption Configurations.
This tool can do things like, help you create new Disruptions given straightforward inputs,
Validate your disruption configurations for structure, explaining a disruption configuration
in english for better understanding, and more.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	_ = rootCmd.Execute()

	defer func() {
		if err := ddmark.CleanupLibraries(); err != nil {
			log.Fatal(err)
		}
	}()
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(func() {
		err := ddmark.InitLibrary(v1beta1.EmbeddedChaosAPI, types.DDMarkChaoslibPrefix)
		if err != nil {
			log.Fatal("didn't init properly")
		}
	})
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.chaosli.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Search config in home directory with name ".chaosli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".chaosli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func DisruptionFromFile(path string) (v1beta1.Disruption, error) {
	parsedSpec := ReadUnmarshalValidate(path)

	return parsedSpec, nil
}

func PrintSeparator() {
	fmt.Println("=======================================================================================================================================")
	// after a separator we should assume there is information we want the user to read before more prints start
	// flooding the terminal. That is why we add this sleep of 2 seconds to give some time for the user to
	// digest the new information before consuming the next
	time.Sleep(2 * time.Second)
}

func ReadUnmarshalValidate(path string) v1beta1.Disruption {
	parsedSpec, err := v1beta1.ReadUnmarshal(path)
	if err != nil {
		log.Fatalf("there were problems reading the disruption at this path: %v", err)
	}

	if err = RunAllValidation(*parsedSpec, path); err != nil {
		log.Fatalf("there were some problems when validating your disruption:\n%v", err)
	}

	return *parsedSpec
}
