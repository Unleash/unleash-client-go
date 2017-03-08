package context

// Context specifies the context in which a feature toggle should be considered
// to be enabled or not.
type Context struct {
	// UserId is the the id of the user.
	UserId string

	// SessionId is the id of the session.
	SessionId string

	// RemoteAddress is the IP address of the machine.
	RemoteAddress string

	// Properties is a map of additional properties.
	Properties map[string]string
}
