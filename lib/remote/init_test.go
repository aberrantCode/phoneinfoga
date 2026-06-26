package remote_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sundowndev/phoneinfoga/v2/lib/filter"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
)

// TestInitScanners verifies the default scanner registry wires up every scanner
// without panicking and exposes each of the new scanners under its public name.
func TestInitScanners(t *testing.T) {
	lib := remote.NewLibrary(filter.NewEngine())

	assert.NotPanics(t, func() {
		remote.InitScanners(lib)
	})

	for _, name := range []string{
		remote.HLR,
		remote.IPQualityScore,
		remote.NANPA,
		remote.SerpAPI,
		"veriphone",
		"abstract",
		"numlookupapi",
	} {
		assert.NotNilf(t, lib.GetScanner(name), "scanner %q should be registered", name)
	}
}
