package server

type Server interface {
	NewHandler(interface{}, ...HandlerOption) Handler

	// registe a Handler to server
	Handle(Handler) error

	// registe middlewares to a handler
	Use(...Plugin) Server

	Start() error
	Stop() error

	GetPaths() []string
}
