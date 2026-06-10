package events_only

import (
	"fmt"

	nobindingshere "github.com/gailsapp/gails/v3/internal/generator/testcases/no_bindings_here"
	"github.com/gailsapp/gails/v3/pkg/application"
)

// SomeClass renders as a TS class.
type SomeClass struct {
	Field  string
	Meadow nobindingshere.HowDifferent[rune]
}

func init() {
	application.RegisterEvent[string]("events_only:string")
	application.RegisterEvent[map[string][]int]("events_only:map")
	application.RegisterEvent[SomeClass]("events_only:class")
	application.RegisterEvent[int]("collision")
	application.RegisterEvent[bool](fmt.Sprintf("events_only:%s%d", "dynamic", 3))
}

func init() {
	application.RegisterEvent[application.Void]("events_only:nodata")
}
