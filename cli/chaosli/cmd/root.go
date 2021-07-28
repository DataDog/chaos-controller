// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	goyaml "github.com/ghodss/yaml"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

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
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(contextCmd)

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
}

func ReadUnmarshalValidate(path string) v1beta1.Disruption {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("finding Absolute Path: %v", err)
	}

	yaml, err := os.Open(filepath.Clean(fullPath))
	if err != nil {
		log.Fatalf("could not open yaml: %v", err)
	}

	yamlBytes, err := ioutil.ReadAll(yaml)
	if err != nil {
		log.Printf("disruption.Get err   #%v ", err)
		os.Exit(1)
	}

	parsedSpec := v1beta1.Disruption{}
	err = goyaml.Unmarshal(yamlBytes, &parsedSpec)

	if err != nil {
		log.Fatalf("unmarshal: %v", err)
	}

	err = parsedSpec.Spec.Validate()

	if err != nil {
		log.Fatalf("there were some problems when validating your disruption: %v", err)
	}

	return parsedSpec
}
