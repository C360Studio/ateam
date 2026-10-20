package bench

import "fmt"

const geminiStableVersion = "0.64.0"
const geminiLegacyVersion = "0.43.0"

func geminiCLIVersion(config *GeminiConfig) string {
	if config.CLIVersion == "" {
		return geminiLegacyVersion
	}
	return config.CLIVersion
}

func validateNewGeminiExecution(config *GeminiConfig) error {
	if config.OutputFormat != "json-schema" {
		return fmt.Errorf("gemini new execution requires output_format json-schema; legacy text format is replay-only")
	}
	if config.CLIVersion != geminiStableVersion {
		return fmt.Errorf("gemini new execution requires explicit cli_version %s; legacy versions are replay-only", geminiStableVersion)
	}
	return nil
}

// The stable CLI defaults missing upstream usage to zero. Such a snapshot
// cannot establish measured zero; keep its source/raw report but leave
// normalized usage unknown. Historical parsing deliberately stays unchanged.
func parseGeminiVersionedOutput(config *GeminiConfig, output string) (AdapterResponse, error) {
	response, err := parseGeminiOutput(output)
	if config.CLIVersion == geminiStableVersion && response.Usage != nil && response.Usage.InputTokens == 0 && response.Usage.OutputTokens == 0 {
		response.Usage = nil
		response.Native.UsageStatus = "partial"
	}
	return response, err
}
