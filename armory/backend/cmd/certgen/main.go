package main

import (
	"flag"
	"fmt"
	"log"

	"armory/internal/certs"
)

func main() {
	dir := flag.String("dir", "certs", "folder for the certificate files")
	flag.Parse()

	if err := certs.Generate(*dir, flag.Args()); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Certificates written to %s\n", *dir)
	fmt.Println("Addresses covered:")
	for _, ip := range certs.LocalIPs() {
		fmt.Println("  ", ip)
	}
	fmt.Printf("Install %s/%s on the tablet once. Keep ca.key private.\n", *dir, certs.CAFile)
}
