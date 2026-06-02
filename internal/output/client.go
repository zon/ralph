package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

var (
	warnColor  = color.New(color.FgYellow)
	errorColor = color.New(color.FgRed)
)

type Client struct {
	out       io.Writer
	err       io.Writer
	debugging bool
}

func NewClient(out, err io.Writer, debugging bool) *Client {
	return &Client{out: out, err: err, debugging: debugging}
}

func (c *Client) Debug(msg string) {
	if !c.debugging {
		return
	}
	fmt.Fprintln(c.out, msg)
}

func (c *Client) Debugf(format string, a ...any) {
	if !c.debugging {
		return
	}
	fmt.Fprintf(c.out, format+"\n", a...)
}

func (c *Client) Info(msg string) {
	fmt.Fprintln(c.out, msg)
}

func (c *Client) Infof(format string, a ...any) {
	fmt.Fprintf(c.out, format+"\n", a...)
}

func (c *Client) Warn(msg string) {
	warnColor.Fprintln(c.out, msg)
}

func (c *Client) Warnf(format string, a ...any) {
	warnColor.Fprintf(c.out, format+"\n", a...)
}

func (c *Client) Error(msg string) {
	errorColor.Fprintln(c.err, msg)
}

func (c *Client) Errorf(format string, a ...any) {
	errorColor.Fprintf(c.err, format+"\n", a...)
}
