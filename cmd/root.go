package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stsysd/launchpack/launchpack"
)

var actionName string

var rootCmd = &cobra.Command{
	Use: "launchpack",
	Run: func(cmd *cobra.Command, args []string) {
		packs := launchpack.LoadPacks()
		if len(packs) == 0 {
			panic("config file not found")
		}
		if actionName == "" {
			action := launchpack.LookUpDefault(packs)
			if action != nil {
				n := action.Exec()
				os.Exit(n)
			}
			panic("default action not found")
		} else {
			action := launchpack.LookUpAction(packs, actionName)
			if action != nil {
				n := action.Exec()
				os.Exit(n)
			}
			panic("action not found")
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&actionName, "action", "a", "", "action name")
}
