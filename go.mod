module github.com/wojciech-malota-wojcik/imagebuilder

go 1.16

replace github.com/ridge/parallel => github.com/wojciech-malota-wojcik/parallel v0.1.2

require (
	github.com/otiai10/copy v1.7.0
	github.com/pkg/errors v0.8.1
	github.com/ridge/must v0.6.0
	github.com/spf13/pflag v1.0.5
	github.com/wojciech-malota-wojcik/build v0.0.0-20210131144749-3ef5b00b908f
	github.com/wojciech-malota-wojcik/buildgo v0.1.1
	github.com/wojciech-malota-wojcik/ioc v1.3.1-0.20210829092813-3edb43f522c7
	github.com/wojciech-malota-wojcik/libexec v0.1.0
	github.com/wojciech-malota-wojcik/logger v0.1.0
	github.com/wojciech-malota-wojcik/run v0.1.2
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1
)
