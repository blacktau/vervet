// Package buildinfo exposes the distribution channel of the running binary,
// set at build time via -ldflags "-X vervet/internal/buildinfo.channel=msstore".
package buildinfo

// ChannelType identifies a Vervet distribution channel.
type ChannelType string

const (
	ChannelGitHub  ChannelType = "github"
	ChannelMSStore ChannelType = "msstore"
)

// channel is overridden at build time via ldflags. Empty / unknown values
// fall back to ChannelGitHub so dev builds and tests behave like the GitHub
// distribution.
var channel string

// Channel returns the current distribution channel.
func Channel() ChannelType {
	switch channel {
	case string(ChannelMSStore):
		return ChannelMSStore
	default:
		return ChannelGitHub
	}
}

// IsMSStore reports whether the binary was built for the Microsoft Store.
func IsMSStore() bool {
	return Channel() == ChannelMSStore
}
