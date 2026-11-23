package main

import (
	// Embed the IANA zone database: the alpine runtime image ships no /usr/share/zoneinfo.
	_ "time/tzdata"

	"github.com/alecthomas/kong"

	"droscher.com/BeerGargoyle/cmd"
)

func main() {
	ctx := kong.Parse(&cmd.CLI, kong.Name("Beer Gargoyle"), kong.Description("BeerGargoyle is a beer cellar management tool."))
	err := ctx.Run(&cmd.Context{Debug: cmd.CLI.Debug})
	ctx.FatalIfErrorf(err)
}
