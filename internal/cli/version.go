package cli

import (
	"encoding/json"
	"fmt"

	"github.com/sharkusmanch/ludusavi-runner/pkg/version"
	"github.com/spf13/cobra"
)

var versionJSON bool

// NewVersionCmd creates the version command.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display detailed version and build information.`,
		RunE:  runVersion,
	}

	cmd.Flags().BoolVar(&versionJSON, "json", false, "output in JSON format")

	return cmd
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := version.Get()

	if versionJSON {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal version info: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Println(info.String())
	}

	return nil
}
