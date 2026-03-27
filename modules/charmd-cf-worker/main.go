// A Dagger module for building, testing, and deploying CHARMD stack Cloudflare
// Workers — Rust crates compiled to WASM via worker-build and tested with Vitest.
//
// The module layers CF worker tooling (Node.js, pnpm, wasm-tools, worker-build)
// on top of the Rust dev container and exposes Build, Test, and Deploy functions.
// pnpm is used as the Node package manager across all CHARMD stack applications.

package main

import (
	"context"
	"dagger/charmd-cf-worker/internal/dagger"
)

const defaultPnpmVersion = "10.30.3"

type CharmdCfWorker struct {
	*dagger.Container
}

// DevContainer creates a CHARMD stack Cloudflare Worker build environment on top
// of the Rust dev container, installing Node.js, pnpm, and the worker-build
// toolchain. Call Build, Test, or Deploy on the returned value.
func (m *CharmdCfWorker) DevContainer(
	//+defaultPath="/"
	source *dagger.Directory,
	// Subdirectory within source that contains the worker package.json (e.g. "hubdash-cf")
	workerDir string,
	//+optional
	toolchainFile *dagger.File,
	// pnpm version to install globally via npm
	//+optional
	//+default="10.30.3"
	pnpmVersion string,
	// Name of the Dagger cache volume used for the pnpm store.
	// If empty, the pnpm store is not cached between runs.
	//+optional
	pnpmCacheVolume string,
) *CharmdCfWorker {
	if pnpmVersion == "" {
		pnpmVersion = defaultPnpmVersion
	}

	ctr := dag.Rust().DevContainer(dagger.RustDevContainerOpts{
		ToolchainFile:     toolchainFile,
		Source:            source,
		ExtraPackages:     []string{"nodejs-22", "npm", "clang", "wasm-tools", "worker-build"},
		ExtraRepositories: []string{"https://moosingin3space.github.io/wolfi-pkgs"},
		ExtraKeyUrls:      []string{"https://moosingin3space.github.io/wolfi-pkgs/melange.rsa.pub"},
	}).Container().
		WithExec([]string{"npm", "install", "-g", "pnpm@" + pnpmVersion}).
		WithWorkdir("/src/" + workerDir).
		WithEnvVariable("CI", "true")

	if pnpmCacheVolume != "" {
		ctr = ctr.WithMountedCache("/root/.local/share/pnpm/store", dag.CacheVolume(pnpmCacheVolume))
	}

	ctr = ctr.WithExec([]string{"pnpm", "install", "--frozen-lockfile"})

	return &CharmdCfWorker{ctr}
}

// Build compiles the Cloudflare Worker in release mode via worker-build.
func (m *CharmdCfWorker) Build(ctx context.Context) (string, error) {
	return m.Container.
		WithExec([]string{"worker-build", "--release"}).
		Stdout(ctx)
}

// Test performs a dry-run wrangler deploy (to validate the build and config)
// followed by the vitest suite.
func (m *CharmdCfWorker) Test(ctx context.Context) (string, error) {
	return m.Container.
		WithExec([]string{"pnpm", "wrangler", "deploy", "--dry-run"}).
		WithExec([]string{"pnpm", "test"}).
		Stdout(ctx)
}

// Deploy deploys the Cloudflare Worker using Wrangler.
func (m *CharmdCfWorker) Deploy(
	ctx context.Context,
	// Cloudflare API token for authentication
	cloudflareApiToken *dagger.Secret,
	// Cloudflare account ID
	cloudflareAccountId *dagger.Secret,
) (string, error) {
	return m.Container.
		WithSecretVariable("CLOUDFLARE_API_TOKEN", cloudflareApiToken).
		WithSecretVariable("CLOUDFLARE_ACCOUNT_ID", cloudflareAccountId).
		WithExec([]string{"pnpm", "run", "deploy"}).
		Stdout(ctx)
}
