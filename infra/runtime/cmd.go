package runtime

import (
	"github.com/spf13/cobra"
	"github.com/wojciech-malota-wojcik/ioc"
	"github.com/wojciech-malota-wojcik/logger"
)

// Args is the list of positional CLI arguments passed to application
type Args []string

// NewCmdFactory returns new CmdFactory
func NewCmdFactory(c *ioc.Container) *CmdFactory {
	return &CmdFactory{
		c: c,
	}
}

// CmdFactory is a wrapper around cobra RunE
type CmdFactory struct {
	c *ioc.Container
}

// Cmd returns function compatible with RunE
func (f *CmdFactory) Cmd(cmdFunc interface{}) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		f.c.Singleton(func() Args {
			return args
		})
		var err error
		f.c.Resolve(func(c *ioc.Container, configRoot ConfigRoot) {
			if !configRoot.VerboseLogging {
				logger.VerboseOff()
			}
			f.c.Call(cmdFunc, &err)
		})
		return err
	}
}
