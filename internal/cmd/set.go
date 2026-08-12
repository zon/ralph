package cmd

type SetCmd struct {
	Config SetConfigCmd `cmd:"" help:"Configure credentials for remote execution"`
}
