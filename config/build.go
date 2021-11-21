package config

import (
	"path/filepath"
	"strings"

	"github.com/wojciech-malota-wojcik/imagebuilder/infra/types"
)

// DefaultTag is used if user specified empty tag list
const DefaultTag types.Tag = "latest"

// NewBuildFactory returns new build config factory
func NewBuildFactory() *BuildFactory {
	return &BuildFactory{}
}

// BuildFactory collects data for build config
type BuildFactory struct {
	// Names is the list of names for corresponding specfiles
	Names []string

	// Tags are used to tag the build
	Tags []string

	// Rebuild forces rebuild of all parent images even if they exist
	Rebuild bool
}

// NewBuild creates build config
func NewBuild(f *BuildFactory, args Args) Build {
	config := Build{
		SpecFiles: args,
		Names:     f.Names,
		Tags:      make(types.Tags, 0, len(f.Tags)),
		Rebuild:   f.Rebuild,
	}

	for i, specFile := range config.SpecFiles {
		if len(config.Names) < i+1 {
			config.Names = append(config.Names, strings.TrimSuffix(filepath.Base(specFile), ".spec"))
		}
	}
	for _, tag := range f.Tags {
		config.Tags = append(config.Tags, types.Tag(tag))
	}
	return config
}

// Build stores configuration for build command
type Build struct {
	// SpecFiles is the list of specfiles to build
	SpecFiles []string

	// Names is the list of names for corresponding specfiles
	Names []string

	// Tags are used to tag the build
	Tags types.Tags

	// Rebuild forces rebuild of all parent images even if they exist
	Rebuild bool
}
