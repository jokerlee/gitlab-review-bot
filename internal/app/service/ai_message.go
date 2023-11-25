package service

import (
	"fmt"
	"strings"
)

func ComposeMessageForAI(title string, description string, diffs []*Diff) string {
	var message strings.Builder
	message.WriteString(fmt.Sprintf("%s\n%s\n", title, description))
	for _, diff := range diffs {
		// ignore go.sum/go.mod
		if !strings.Contains(diff.OldPath, "go.sum") && !strings.Contains(diff.OldPath, "go.mod") {
			message.WriteString(diff.Content)
		}
	}
	return message.String()
}
