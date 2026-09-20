package sys

import (
	"github.com/mlange-42/ark-tools/system"
	"github.com/pwn-model/pwn/config"
)

// init registers system types owned by imported packages (as opposed to
// this package's own systems, which self-register in their own files),
// under the same config type-name registry.
func init() {
	config.Register[system.FixedTermination]()
}
