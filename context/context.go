package context

type Context struct {
	UserId        string
	SessionId     string
	RemoteAddress string
	Properties    map[string]string
}
