package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/otiai10/copy"
	"github.com/ridge/must"
	"github.com/wojciech-malota-wojcik/imagebuilder/infra/storage"
	"github.com/wojciech-malota-wojcik/libexec"
)

const manifestFile = "manifest.json"

type cloneFromFn func(srcImageName string) (ImageManifest, error)

// NewBuilder creates new image builder
func NewBuilder(repo *Repository, storage storage.Driver) *Builder {
	return &Builder{
		repo:    repo,
		storage: storage,
	}
}

// Builder builds images
type Builder struct {
	repo    *Repository
	storage storage.Driver
}

// Build builds images
func (b *Builder) Build(ctx context.Context, img *Descriptor) (imgBuild *ImageBuild, retErr error) {
	if err := b.storage.Drop(img.Name()); err != nil {
		return nil, err
	}
	path, err := ioutil.TempDir("/tmp", "imagebuilder-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.Remove(path); retErr == nil {
			retErr = err
		}
	}()

	var imgUnmount storage.UnmountFn
	defer func() {
		if err := umount(path); err != nil && retErr == nil {
			retErr = err
		}
		if imgUnmount != nil {
			if err := imgUnmount(); err != nil && retErr == nil {
				retErr = err
			}
		}
	}()
	if err := umount(path); err != nil {
		return nil, err
	}

	var base bool
	if strings.HasPrefix(img.Name(), "fedora:") {
		parts := strings.SplitN(img.Name(), ":", 2)
		if len(parts) != 2 || parts[1] == "" {
			return nil, errors.New("no tag provided for base image")
		}
		fedoraRelease := parts[1]

		var err error
		imgUnmount, err = b.storage.Mount(img.Name(), path)
		if err != nil {
			return nil, err
		}

		if err := libexec.Exec(ctx,
			exec.Command("dnf", "install", "-y", "--installroot="+path, "--releasever="+fedoraRelease,
				"--setopt=install_weak_deps=False", "--setopt=keepcache=False", "--nodocs",
				"dnf", "langpacks-en")); err != nil {
			return nil, err
		}
	}

	build := newImageBuild(base, path, func(srcImageName string) (ImageManifest, error) {
		err = b.storage.Clone(srcImageName, img.Name())
		if err != nil && errors.Is(err, storage.ErrSourceImageDoesNotExist) {
			if baseImage := b.repo.Retrieve(srcImageName); baseImage != nil {
				if _, err := b.Build(ctx, baseImage); err != nil {
					return ImageManifest{}, err
				}
				err = b.storage.Clone(srcImageName, img.Name())
			} else {
				return ImageManifest{}, fmt.Errorf("image %s does not exist in repository", srcImageName)
			}
		}
		if err != nil {
			return ImageManifest{}, err
		}

		imgUnmount, err = b.storage.Mount(img.Name(), path)
		if err != nil {
			return ImageManifest{}, err
		}

		manifestRaw, err := ioutil.ReadFile(filepath.Join(path, manifestFile))
		if err != nil {
			return ImageManifest{}, err
		}
		if err := os.Remove(filepath.Join(path, manifestFile)); err != nil {
			return ImageManifest{}, err
		}

		var manifest ImageManifest
		if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
			return ImageManifest{}, err
		}

		if err := syscall.Mount("/dev", filepath.Join(path, "dev"), "", syscall.MS_BIND, ""); err != nil {
			return ImageManifest{}, err
		}
		if err := syscall.Mount("/proc", filepath.Join(path, "proc"), "", syscall.MS_BIND, ""); err != nil {
			return ImageManifest{}, err
		}
		if err := syscall.Mount("/sys", filepath.Join(path, "sys"), "", syscall.MS_BIND, ""); err != nil {
			return ImageManifest{}, err
		}
		return manifest, nil
	})

	for _, cmd := range img.commands {
		if err := cmd.execute(ctx, build); err != nil {
			return nil, err
		}
	}

	if err := ioutil.WriteFile(filepath.Join(path, manifestFile), must.Bytes(json.Marshal(build.Manifest())), 0o444); err != nil {
		return nil, err
	}

	return build, nil
}

func umount(imgPath string) error {
	mountsRaw, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return err
	}
	for _, mount := range strings.Split(string(mountsRaw), "\n") {
		props := strings.SplitN(mount, " ", 3)
		if len(props) < 2 {
			// last empty line
			break
		}
		mountpoint := props[1]
		if !strings.HasPrefix(mountpoint, imgPath+"/") {
			continue
		}
		if err := syscall.Unmount(mountpoint, 0); err != nil {
			return err
		}
	}
	return nil
}

// ImageManifest contains info about built image
type ImageManifest struct {
	Params []string
}

func newImageBuild(base bool, path string, cloneFn cloneFromFn) *ImageBuild {
	return &ImageBuild{
		cloneFn: cloneFn,
		base:    base,
		path:    path,
	}
}

// ImageBuild represents build of an image
type ImageBuild struct {
	cloneFn cloneFromFn

	base     bool
	path     string
	manifest ImageManifest
}

// Path returns path to directory where image is created
func (b *ImageBuild) Path() string {
	return b.path
}

// Manifest returns image manifest
func (b *ImageBuild) Manifest() ImageManifest {
	return b.manifest
}

// from is a handler for FROM
func (b *ImageBuild) from(cmd *fromCommand) error {
	if b.base {
		return errors.New("command FROM is forbidden for base image")
	}
	manifest, err := b.cloneFn(cmd.imageName)
	if err != nil {
		return err
	}
	b.manifest.Params = manifest.Params
	return nil
}

// params sets kernel params for image
func (b *ImageBuild) params(cmd *paramsCommand) error {
	b.manifest.Params = append(b.manifest.Params, cmd.params...)
	return nil
}

// copy is a handler for COPY
func (b *ImageBuild) copy(cmd *copyCommand) error {
	return copy.Copy(cmd.from, filepath.Join(b.path, cmd.to), copy.Options{PreserveTimes: true, PreserveOwner: true})
}

// run is a handler for RUN
func (b *ImageBuild) run(ctx context.Context, cmd *runCommand) (retErr error) {
	root, err := os.Open("/")
	if err != nil {
		return err
	}
	defer root.Close()

	curr, err := os.Open(".")
	if err != nil {
		return err
	}
	defer curr.Close()

	if err := syscall.Chroot(b.path); err != nil {
		return err
	}
	if err := os.Chdir("/"); err != nil {
		return err
	}

	defer func() {
		defer func() {
			if err := curr.Chdir(); err != nil && retErr == nil {
				retErr = err
			}
		}()

		if err := root.Chdir(); err != nil {
			if retErr == nil {
				retErr = err
			}
			return
		}
		if err := syscall.Chroot("."); err != nil {
			if retErr == nil {
				retErr = err
			}
			return
		}
	}()

	return libexec.Exec(ctx, exec.Command("sh", "-c", cmd.command))
}
