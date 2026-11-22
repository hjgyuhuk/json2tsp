package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"json2tsp/convert"
)

// NewRootCommand creates the root CLI command for json2tsp.
func NewRootCommand() *cobra.Command {
	var opts convert.Options

	rootCmd := &cobra.Command{
		Use:           "json2tsp [file]",
		Short:         "Convert JSON from a file or stdin into TypeSpec definitions",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeConvertCommand(cmd, args, opts)
		},
	}

	rootCmd.Flags().StringVarP(&opts.RootName, "name", "n", "Root", "root TypeSpec model or alias name")
	rootCmd.Flags().BoolVar(&opts.Inline, "inline", false, "inline nested objects instead of extracting named models")

	return rootCmd
}

func executeConvertCommand(cmd *cobra.Command, args []string, opts convert.Options) error {
	input, closer, err := openInputReader(cmd, args)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	output, err := convert.ConvertJSONToTypeSpec(input, opts)
	if err != nil {
		return err
	}

	if _, err := io.WriteString(cmd.OutOrStdout(), output); err != nil {
		return fmt.Errorf("write typespec output: %w", err)
	}
	return nil
}

func openInputReader(cmd *cobra.Command, args []string) (io.Reader, io.Closer, error) {
	if len(args) == 0 || args[0] == "-" {
		return cmd.InOrStdin(), nil, nil
	}

	file, err := os.Open(args[0])
	if err != nil {
		return nil, nil, fmt.Errorf("open json file %q: %w", args[0], err)
	}
	return file, file, nil
}
