package main

import (
	"context"
	"fmt"
	"net/http"

	uspsApi "github.com/pistolricks/ShippingApi/internal/usps"
)

func (app *application) handleShippingRates(w http.ResponseWriter, r *http.Request) {
	// Create the client (already provided by app helper):
	ratesClient := app.postalRatesClient()

	// Build request according to USPS Domestic Prices v3 API:
	reqBody := uspsApi.DomesticBaseRatesRequest{
		OriginZIPCode:                 "30301",
		DestinationZIPCode:            "90210",
		Weight:                        1.0, // ounces, example
		Length:                        2.0,
		Width:                         3.0,
		Height:                        3.0,
		MailClass:                     "USPS_GROUND_ADVANTAGE",
		ProcessingCategory:            "MACHINABLE",
		RateIndicator:                 "SP",
		DestinationEntryFacilityType:  "NONE",
		PriceType:                     "COMMERCIAL",
		MailingDate:                   "2025-12-20",
		AccountType:                   "MID",
		AccountNumber:                 "903950522",
		HasNonstandardCharacteristics: false,
		// Add other USPS-supported params as needed (serviceType, machinable, dimensions, etc.)
	}

	// Call the endpoint
	fmt.Printf("%s", reqBody)

	res, err := ratesClient.SearchDomesticBaseRates(context.Background(), reqBody)
	if err != nil {
		app.badRequestResponse(w, r, err)
	}

	fmt.Printf("response %s", res)

	err = app.writeJSON(w, 200, envelope{"rates": res}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
