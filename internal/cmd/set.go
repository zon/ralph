package cmd

type SetCmd struct {
	Remote SetRemoteCmd `cmd:"" help:"Configure credentials for remote execution"`
}
