// CI pipeline for testing the Rust Dagger module.
package main

import (
	"context"
	"dagger/ci/internal/dagger"

	"golang.org/x/sync/errgroup"
)

type Ci struct{}

// testdata returns the bundled Rust fixture project used by all tests.
func testdata() *dagger.Directory {
	return dag.CurrentModule().Source().Directory("testdata")
}

// All runs every test scenario in parallel.
func (m *Ci) All(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error { return m.TestStandalone(ctx) })
	eg.Go(func() error { return m.TestWithExtraPackages(ctx) })
	eg.Go(func() error { return m.TestWithExtraRepositories(ctx) })
	return eg.Wait()
}

// TestStandalone validates basic usage of the Rust module without any extra options.
func (m *Ci) TestStandalone(ctx context.Context) error {
	r := dag.Rust().DevContainer(dagger.RustDevContainerOpts{Source: testdata()})
	if _, err := r.CargoCheck(ctx); err != nil {
		return err
	}
	if _, err := r.CargoFmtCheck(ctx); err != nil {
		return err
	}
	if _, err := r.CargoClippy(ctx); err != nil {
		return err
	}
	return nil
}

// TestWithExtraPackages validates that additional Wolfi packages can be
// installed alongside the default Rust toolchain.
func (m *Ci) TestWithExtraPackages(ctx context.Context) error {
	r := dag.Rust().DevContainer(dagger.RustDevContainerOpts{
		Source:        testdata(),
		ExtraPackages: []string{"git", "cmake"},
	})
	_, err := r.CargoCheck(ctx)
	return err
}

// TestWithExtraRepositories validates that extra APK repositories and their
// signing keys can be provided. This example adds the Chainguard extras
// repository so that packages outside the base Wolfi set are available.
func (m *Ci) TestWithExtraRepositories(ctx context.Context) error {
	r := dag.Rust().DevContainer(dagger.RustDevContainerOpts{
		Source:            testdata(),
		ExtraRepositories: []string{"https://packages.cgr.dev/extras"},
		ExtraKeyUrls:      []string{"https://packages.cgr.dev/extras/chainguard-extras.rsa.pub"},
		// Install an extra package that lives in the extras repository to
		// prove the repo configuration is actually used.
		ExtraPackages: []string{"git"},
	})
	_, err := r.CargoCheck(ctx)
	return err
}
