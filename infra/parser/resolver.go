package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/wojciech-malota-wojcik/imagebuilder/infra/description"
	"github.com/wojciech-malota-wojcik/ioc"
)

// NewResolvingParser returns new auto resolving parser
func NewResolvingParser(c *ioc.Container) Parser {
	return &resolvingParser{
		c: c,
	}
}

type resolvingParser struct {
	c *ioc.Container
}

// Parse parses file using resolver matching the extension of a file
func (p *resolvingParser) Parse(filePath string) ([]description.Command, error) {
	names := map[string]bool{}
	for _, name := range p.c.Names((*Parser)(nil)) {
		names[name] = true
	}

	var ext string
	if i := strings.LastIndex(filePath, "."); i >= 0 {
		ext = filePath[i+1:]
	}
	if ext == "" {
	loop:
		for _, e := range p.c.Names((*Parser)(nil)) {
			f := filePath + "." + e
			info, err := os.Stat(f)
			switch {
			case err != nil && !os.IsNotExist(err):
				return nil, err
			case err == nil && !info.IsDir():
				filePath = f
				ext = e
				break loop
			}
		}
	}

	if !names[ext] {
		return nil, fmt.Errorf("parser not found for file %s", filePath)
	}

	var parser Parser
	p.c.ResolveNamed(ext, &parser)
	return parser.Parse(filePath)
}
