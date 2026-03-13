// Package main — точка входа клиентского CLI-приложения GophKeeper.
package main

import "fmt"

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
)

func main() {
	fmt.Printf("GophKeeper Client %s (built: %s)\n", buildVersion, buildDate)
}
