package cmd

import (
	"encoding/json"
	"os"
)

// jsonEncode writes v as indented JSON to stdout.
func jsonEncode(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func init() {
	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "skip the confirmation prompt")
	execCmd.Flags().StringVar(&execEnv, "env", "", "environment variable name to inject (default: the entry name)")
}
