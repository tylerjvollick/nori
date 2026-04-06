package main

import "github.com/tylerjvollick/nori/cmd"

func main() {
	cmd.MigrationsFS = MigrationsFS
	cmd.Execute()
}
