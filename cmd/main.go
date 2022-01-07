package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ApogeeNetworking/sandvine"
	"github.com/subosito/gotenv"
)

var srpHost, srpUser, srpPass string

func init() {
	gotenv.Load()
	if srpHost == "" || srpUser == "" || srpPass == "" {
		srpHost = os.Getenv("SRP_HOST")
		srpUser = os.Getenv("SRP_USER")
		srpPass = os.Getenv("SRP_PASS")
	}
}

func main() {
	sv := sandvine.NewService(srpHost, srpUser, srpPass)
	err := sv.Client.Connect(2)
	if err != nil {
		log.Fatal(err)
	}
	defer sv.Client.Disconnect()

	// sv.TriageSrp()
	dbstat, _ := sv.ShowDbServStatus()
	fmt.Println(dbstat)
}
