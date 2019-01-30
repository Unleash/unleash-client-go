package context

// Context specifies the context in which a feature toggle should be considered
// to be enabled or not.
type Context struct {
	// UserID is the the id of the user.
	UserID string

	// SessionID is the id of the session.
	SessionID string

	// RemoteAddress is the IP address of the machine.
	RemoteAddress string

	// Properties is a map of additional properties.
	Properties map[string]string
}
