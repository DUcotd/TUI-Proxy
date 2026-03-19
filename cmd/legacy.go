package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func legacyRunner(oldPath, newPath string, runner func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		printLegacyCommandNotice(oldPath, newPath)
		return runner(cmd, args)
	}
}

func printLegacyCommandNotice(oldPath, newPath string) {
	fmt.Printf("⚠️  '%s' 已迁移到 '%s'；当前入口仅为兼容保留，后续版本将移除。\n\n", oldPath, newPath)
}
