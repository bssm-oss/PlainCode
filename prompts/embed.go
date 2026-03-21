package prompttpl

import _ "embed"

// BuildSystem is the embedded system prompt used for build executions.
//
//go:embed build-system.txt
var BuildSystem string
