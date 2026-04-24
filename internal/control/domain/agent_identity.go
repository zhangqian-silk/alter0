package domain

import "strings"

const BuiltinTravelAgentID = "travel"

func IsTravelAgentID(raw string) bool {
	return strings.EqualFold(strings.TrimSpace(raw), BuiltinTravelAgentID)
}
