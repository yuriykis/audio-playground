//go:build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	clientBin = "bin/client"

	// nolibopusfile skips the Ogg/Opus file reader in hraban/opus, so only libopus is needed.
	buildTags = "nolibopusfile"
)

// Build compiles the client into bin/.
func Build() error {
	return sh.RunV("go", "build", "-tags", buildTags, "-o", clientBin, "./cmd/client")
}

// Run builds the client and starts it.
func Run() error {
	mg.Deps(Build)
	return sh.RunV("./" + clientBin)
}

// Lint runs golangci-lint on the whole module and the magefile.
func Lint() error {
	return sh.RunV("golangci-lint", "run", "--build-tags", "mage,"+buildTags, "./...")
}
