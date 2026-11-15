package jsmodules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Node treats `\temp\x` and `/temp/x` as absolute on the current drive, so
// they must not be joined onto the script's directory (issue #369).
func TestResolve_RootedPathsUseScriptDrive(t *testing.T) {
	base := `D:\scripts\mongo`
	assert.Equal(t, `D:\temp\test.csv`, resolve(base, `\temp\test.csv`))
	assert.Equal(t, `D:/temp/test.csv`, resolve(base, `/temp/test.csv`))
	assert.Equal(t, `C:\temp\test.csv`, resolve(base, `C:\temp\test.csv`))
	assert.Equal(t, `C:/temp/test.csv`, resolve(base, `C:/temp/test.csv`))
	assert.Equal(t, `\\server\share\x.csv`, resolve(base, `\\server\share\x.csv`))
	assert.Equal(t, `D:\scripts\mongo\data\x.csv`, resolve(base, `data\x.csv`))
}

func TestIsAbs_MatchesNodeOnWindows(t *testing.T) {
	assert.True(t, isAbs(`\temp`))
	assert.True(t, isAbs(`/temp`))
	assert.True(t, isAbs(`C:\temp`))
	assert.True(t, isAbs(`\\server\share`))
	assert.False(t, isAbs(`temp\x`))
	assert.False(t, isAbs(`C:temp`))
}
