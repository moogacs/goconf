package types

type Status byte

// Each state represents package status
// The package enums matches the statuses for http://man7.org/linux/man-pages/man1/dpkg-query.1.html
const (
	StatusInstalled    = 'i'
	StatusNotInstalled = 'n'

	StatusRestarted = iota
	StatusStarted
)

// APT is a apt/dpkg package or service
type APT struct {
	Name   string
	Status Status
	User   string
}

// Service return bool if the apt is a service
func (p *APT) Service() bool {
	return p.Status == StatusStarted || p.Status == StatusRestarted
}
