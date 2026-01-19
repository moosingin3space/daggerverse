// A set of functions that can be used to build Rust easily.

package main

import (
	"context"
	"dagger/rust/internal/dagger"
)

type Rust struct {
	*dagger.Container
	source *dagger.Directory
}

// Creates a Rust development environment in a container.
func (m *Rust) DevContainer(
	//+defaultPath="/"
	source *dagger.Directory,
	//+optional
	toolchainFile *dagger.File,
	//+optional
	extraPackages []string,
) *Rust {
	packages := []string{
		"rustup",
		"build-base",
		"openssl-dev",
		"pkgconf",
		"curl",
	}
	if extraPackages != nil {
		packages = append(packages, extraPackages...)
	}
	// TODO try not to hardcode
	// TODO key these caches appropriately
	ctr := dag.Wolfi().
		Container(dagger.WolfiContainerOpts{
			Packages: packages,
		}).
		WithEnvVariable("CARGO_HOME", "/usr/local/cargo").
		WithEnvVariable("RUSTUP_HOME", "/usr/local/rustup").
		WithMountedCache("/usr/local/cargo", dag.CacheVolume("cargo-home")).
		WithMountedCache("/usr/local/rustup", dag.CacheVolume("rustup-home")).
		WithExec([]string{"rustup-init", "-y", "--default-toolchain", "none"}).
		WithEnvVariable("PATH", "/usr/local/cargo/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin").
		WithWorkdir("/src")
	if toolchainFile != nil {
		ctr = ctr.WithMountedFile("/src/rust-toolchain.toml", toolchainFile).
			WithExec([]string{"rustup", "toolchain", "install"})
	} else {
		// Default to the latest stable toolchain
		ctr = ctr.WithExec([]string{"rustup", "toolchain", "install", "stable"})
	}
	ctr = ctr.WithMountedDirectory("/src", source).
		WithMountedCache("/src/target", dag.CacheVolume("target"))
	return &Rust{
		ctr,
		source,
	}
}

func (m *Rust) CargoCheck(
	ctx context.Context,
) (string, error) {
	return m.Container.
		WithExec([]string{"cargo", "check", "--all", "--all-targets"}).
		Stdout(ctx)
}

func (m *Rust) CargoFmtCheck(
	ctx context.Context,
) (string, error) {
	return m.Container.
		WithExec([]string{"rustup", "component", "add", "rustfmt"}).
		WithExec([]string{"cargo", "fmt", "--all", "--check"}).
		Stdout(ctx)
}

func (m *Rust) CargoFmtFix() *dagger.Changeset {
	generated := m.Container.
		WithExec([]string{"rustup", "component", "add", "rustfmt"}).
		WithExec([]string{"cargo", "fmt", "--all"}).
		Directory("/src")
	return generated.Changes(m.source)
}
