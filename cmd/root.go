package cmd

type Context struct {
	Debug bool
}

var CLI struct {
	Debug bool `help:"Enable debug mode"`

	Serve   ServeCmd   `cmd:"" default:"1"                    help:"Run the server"`
	Migrate MigrateCmd `cmd:"" help:"Run database migrations"`
}
