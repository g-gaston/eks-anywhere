package framework

const (
	Ubuntu2004Version    = "2004" // corresponds to Ubuntu 20.04
	Ubuntu2204Version    = "2204" // corresponds to Ubuntu 22.04
	RedHat8Version       = "8"    // corresponds to RedHat 8
	Bottlerocket1Version = "1"    // corresponds to Bottlerocket 1
)

var defaultOSVersions = map[string]string{
	"ubuntu":       Ubuntu2004Version,
	"redhat":       RedHat8Version,
	"bottlerocket": Bottlerocket1Version,
}
