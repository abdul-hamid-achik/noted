package theme

import _ "embed"

// GlamourJSON is a Nord-matched Glamour style for markdown preview (used via
// glamour.WithStylesFromJSONBytes). See nord_glamour.json.
//
//go:embed nord_glamour.json
var GlamourJSON []byte
