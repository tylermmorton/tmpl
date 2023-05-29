/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tmpl",
	Short: "tmpl is a html/template toolchain",
	Long:  `https://github.com/tylermmorton/tmpl`,
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func init() {}

// Converts snake_case to camelCase
func toCamelCase(inputUnderScoreStr string) (camelCase string) {
	flag := false
	for k, v := range inputUnderScoreStr {
		if k == 0 {
			camelCase = strings.ToUpper(string(inputUnderScoreStr[0]))
		} else {
			if flag {
				camelCase += strings.ToUpper(string(v))
				flag = false
			} else {
				if v == '-' || v == '_' {
					flag = true
				} else {
					camelCase += string(v)
				}
			}
		}
	}
	return
}
