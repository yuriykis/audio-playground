//go:build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	binPath = "bin/audio-playground"

	// nolibopusfile skips the Ogg/Opus file reader in hraban/opus, so only libopus is needed.
	buildTags = "nolibopusfile"
)

// Build compiles the app into bin/.
func Build() error {
	return sh.RunV("go", "build", "-tags", buildTags, "-o", binPath, ".")
}

// Run builds the app and starts it.
func Run() error {
	mg.Deps(Build)
	return sh.RunV("./" + binPath)
}

// Lint runs golangci-lint on the app and the magefile.
func Lint() error {
	return sh.RunV("golangci-lint", "run", "--build-tags", "mage,"+buildTags, "./...")
}
