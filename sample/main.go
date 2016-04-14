package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/pivotalservices/opsmanclient"
)

func main() {
	opsmanUser := flag.String("u", "admin", "Ops Manager User")
	opsmanPassword := flag.String("p", "password", "Ops Manager Password")
	opsmanURL := flag.String("url", "https://192.168.200.10", "Ops Manager URL")
	saveFile := flag.String("f", "installation.json", "Save deployment to JSON file (defaults to installation.json)")
	flag.Parse()

	opsman := opsmanclient.New(*opsmanURL, *opsmanUser, *opsmanPassword)

	// Check we are using a supported Ops Man
	version, err := opsman.GetAPIVersion()
	if err != nil {
		log.Fatal(err)
	}

	opsmanclient.ValidateAPIVersion(version)
	if err != nil {
		log.Fatal(err)
	}

	// Get installation settings from Ops Man foundation
	installation, err := opsman.GetInstallationSettings()
	if err != nil {
		log.Fatal(err)
	}

	products, err := opsman.GetProducts()
	if err != nil {
		log.Fatal(err)
	}

	cf, err := opsman.GetCFDeployment(installation, products)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Your CF deployment release:", cf.Release)

	d, err := json.Marshal(installation)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(*saveFile, d, 0644)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Your installation was saved to", *saveFile)
}
