package usps

import (
	"context"
	"fmt"
	"log"

	"github.com/my-eq/go-usps"
	"github.com/my-eq/go-usps/models"
)

func StandardizedAddress(client *usps.Client, addressRequest *models.AddressRequest) (*models.AddressResponse, error) {

	req := &models.AddressRequest{
		StreetAddress: "1578 Topeka Avenue",
		City:          "Placentia",
		State:         "CA",
	}

	resp, err := client.GetAddress(context.Background(), req)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Standardized: %s, %s, %s %s\n",
		resp.Address.StreetAddress,
		resp.Address.City,
		resp.Address.State,
		resp.Address.ZIPCode)

	return resp, err
}
