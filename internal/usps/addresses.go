package usps

import (
	"context"
	"fmt"
	"log"

	"github.com/my-eq/go-usps"
	"github.com/my-eq/go-usps/models"
)

func StandardizedAddress(client *usps.Client, address *models.AddressRequest) (*models.AddressResponse, error) {

	req := &models.AddressRequest{
		Firm:             address.Firm,
		StreetAddress:    address.StreetAddress,
		SecondaryAddress: address.SecondaryAddress,
		City:             address.City,
		State:            address.State,
		Urbanization:     address.Urbanization,
		ZIPCode:          address.ZIPCode,
		ZIPPlus4:         address.ZIPPlus4,
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
