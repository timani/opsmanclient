package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/pivotalservices/datadog-dashboard-gen/opsman"
)

func main() {
	opsmanUser := flag.String("u", "admin", "Ops Manager User")
	opsmanPassword := flag.String("p", "password", "Ops Manager Password")
	opsmanIP := flag.String("ip", "192.168.200.10", "Ops Manager IP")
	saveFile := flag.String("f", "installation.json", "Save deployment to JSON file (defaults to installation.json)")
	flag.Parse()

	opsmanClient := opsman.New(*opsmanIP, *opsmanUser, *opsmanPassword)

	// Check we are using a supported Ops Man
	err := opsman.ValidateAPIVersion(opsmanClient)
	if err != nil {
		log.Fatal(err)
	}

	// Get installation settings from Ops Man foundation
	installation, err := opsmanClient.GetInstallationSettings()
	if err != nil {
		log.Fatal(err)
	}

	products, err := opsmanClient.GetProducts()
	if err != nil {
		log.Fatal(err)
	}

	deployment, err := opsmanClient.GetCFDeployment(installation, products)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Your deployment is using CF release:", deployment.Release)

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
