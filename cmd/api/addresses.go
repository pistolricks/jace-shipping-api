package main

import (
	"net/http"

	"github.com/my-eq/go-usps/models"
	uspsApi "github.com/pistolricks/ShippingApi/internal/usps"
)

func (app *application) handleStandardAddress(w http.ResponseWriter, r *http.Request) {

	var input struct {
		Firm             string `json:"firm,omitempty"`
		StreetAddress    string `json:"streetAddress"`
		SecondaryAddress string `json:"secondaryAddress,omitempty"`
		City             string `json:"city,omitempty"`
		State            string `json:"state"`
		Urbanization     string `json:"urbanization,omitempty"`
		ZIPCode          string `json:"ZIPCode,omitempty"`
		ZIPPlus4         string `json:"ZIPPlus4,omitempty"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Standardize an address
	req := &models.AddressRequest{
		Firm:             input.Firm,
		StreetAddress:    input.StreetAddress,
		SecondaryAddress: input.SecondaryAddress,
		City:             input.City,
		State:            input.State,
		Urbanization:     input.Urbanization,
		ZIPCode:          input.ZIPCode,
		ZIPPlus4:         input.ZIPPlus4,
	}

	resp, err := uspsApi.StandardizedAddress(app.postalClient(), req)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"Street":  resp.Address.StreetAddress,
		"City":    resp.Address.City,
		"State":   resp.Address.State,
		"ZIPCode": resp.Address.ZIPCode,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
