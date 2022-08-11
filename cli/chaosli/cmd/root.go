// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// Version will be set with the -ldflags option at compile time
var Version = "v0"
var APILibPath = fmt.Sprintf("chaosli-api-lib/v1beta1/%v", Version)
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

	defer cleanupLibrary()
}

func init() {
	cobra.OnInitialize(initConfig, initLibrary)

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

// initLibrary copies the binary-embedded disruption API into a custom folder in GOPATH
func initLibrary() {
	if _, isGoInstalled := os.LookupEnv("GOPATH"); !isGoInstalled {
		log.Fatal("Setup error: please make sure go (1.18 or higher) is installed and the GOPATH is set")
	}

	if err := os.Setenv("GO111MODULE", "off"); err != nil {
		log.Fatal(err)
	}

	folderPath := fmt.Sprintf("%v/src/%v/", os.Getenv("GOPATH"), APILibPath)

	if err := os.MkdirAll(folderPath, 0750); err != nil {
		log.Fatal(err)
	}

	err := fs.WalkDir(v1beta1.EmbeddedChaosAPI, ".",
		// this function is executed for every file found within the binary-embedded folder
		// it copies every files to another location on the computer through io.Copy
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			fin, err := fs.ReadFile(v1beta1.EmbeddedChaosAPI, path)
			if err != nil {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			fout, err := os.Create(folderPath + info.Name())
			if err != nil {
				return err
			}

			if _, err = fout.Write(fin); err != nil {
				return err
			}

			if err = fout.Close(); err != nil {
				return err
			}

			return nil
		})

	if err != nil {
		log.Fatal(err)
	}
}

func cleanupLibrary() {
	cleanupPath := fmt.Sprintf("%v/src/chaosli-api-lib", os.Getenv("GOPATH"))
	if os.RemoveAll(cleanupPath) != nil {
		log.Println("couldn't clean up API located at " + fmt.Sprintf("%v/src/chaosli-api-lib", os.Getenv("GOPATH")))
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
