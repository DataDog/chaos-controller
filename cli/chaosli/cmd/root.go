// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/markbates/pkger"
	goyaml "sigs.k8s.io/yaml"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const APILIBPATH string = "chaosli-api-lib/v1beta1"

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
	if _, isGoInstalled := os.LookupEnv("GOPATH"); !isGoInstalled {
		log.Fatal("please make sure go (1.11 or higher) is installed and the GOPATH is set")
	}

	os.Setenv("GO111MODULE", "off")

	folderPath := os.Getenv("GOPATH") + "/src/" + APILIBPATH + "/"
	err := os.MkdirAll(folderPath, 0777)
	if err != nil {
		log.Fatal(err)
	}

	pkger.Walk("github.com/DataDog/chaos-controller:/api/v1beta1", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fin, err := pkger.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		defer fin.Close()

		fout, err := os.Create(folderPath + info.Name())
		if err != nil {
			log.Fatal(err)
		}
		defer fout.Close()

		_, err = io.Copy(fout, fin)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	})

	cobra.OnInitialize(initConfig)

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
		log.Fatalf("disruption.Get err   #%v ", err)
	}

	parsedSpec := v1beta1.Disruption{}

	if err = goyaml.UnmarshalStrict(yamlBytes, &parsedSpec); err != nil {
		log.Fatalf("unmarshal: %v", err)
	}

	if err = RunAllValidation(parsedSpec, path); err != nil {
		log.Fatalf("there were some problems when validating your disruption:\n%v", err)
	}

	return parsedSpec
}
