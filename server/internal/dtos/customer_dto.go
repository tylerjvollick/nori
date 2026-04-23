package dtos

import "github.com/tylerjvollick/nori/internal/models"

// CustomerResponse is the API response for a customer.
type CustomerResponse struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

// CustomerResponseFromModel converts a Customer model to a CustomerResponse.
func CustomerResponseFromModel(c *models.Customer) CustomerResponse {
	return CustomerResponse{
		ID:    c.ID.String(),
		Name:  c.Name,
		Email: c.Email,
		Phone: c.Phone,
	}
}
