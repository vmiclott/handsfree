// Package cmd defines the handsfree CLI
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "handsfree",
	Short: "Control your computer without using hands",
	Long:  "HandsFree is a hands-free voice control application that transcribes audio input and executes actions based on the recognized speech.",
	Run:   run,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}

func run(cmd *cobra.Command, args []string) {
	fmt.Println("Hello world.")
}
