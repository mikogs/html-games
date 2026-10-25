// Package htmlgames holds the files of the repository that are built into
// the server.
package htmlgames

import "embed"

// Questions are the sets of questions written by hand, one
// questions/<slug>.json per set, see internal/catalog.
//
//go:embed questions/*.json
var Questions embed.FS
