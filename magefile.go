//go:build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	clientBin = "bin/client"
	serverBin = "bin/server"

	// nolibopusfile skips the Ogg/Opus file reader in hraban/opus, so only libopus is needed.
	buildTags = "nolibopusfile"
)

// Build compiles the client and the server into bin/.
func Build() error {
	if err := sh.RunV("go", "build", "-tags", buildTags, "-o", clientBin, "./cmd/client"); err != nil {
		return err
	}
	return sh.RunV("go", "build", "-tags", buildTags, "-o", serverBin, "./cmd/server")
}

// Run builds everything and starts the client.
func Run() error {
	mg.Deps(Build)
	return sh.RunV("./" + clientBin)
}

// Server builds everything and starts the server.
func Server() error {
	mg.Deps(Build)
	return sh.RunV("./" + serverBin)
}

// Lint runs golangci-lint on the whole module and the magefile.
func Lint() error {
	return sh.RunV("golangci-lint", "run", "--build-tags", "mage,"+buildTags, "./...")
}
