
package tickets

import "testing"

func TestCreateTicketRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request CreateTicketRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: CreateTicketRequest{
				Subject:       "Payment issue",
				Description:   "My payment was unsuccessful.",
				CustomerName:  "John Smith",
				CustomerEmail: "john@example.com",
				Channel:       "web",
			},
			wantErr: false,
		},
		{
			name: "missing subject",
			request: CreateTicketRequest{
				Description:   "Payment failed.",
				CustomerName:  "John Smith",
				CustomerEmail: "john@example.com",
				Channel:       "web",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			request: CreateTicketRequest{
				Subject:       "Payment issue",
				Description:   "Payment failed.",
				CustomerName:  "John Smith",
				CustomerEmail: "invalid-email",
				Channel:       "web",
			},
			wantErr: true,
		},
		{
			name: "invalid channel",
			request: CreateTicketRequest{
				Subject:       "Payment issue",
				Description:   "Payment failed.",
				CustomerName:  "John Smith",
				CustomerEmail: "john@example.com",
				Channel:       "fax",
			},
			wantErr: true,
		},
		{
			name: "missing customer name",
			request: CreateTicketRequest{
				Subject:       "Payment issue",
				Description:   "Payment failed.",
				CustomerEmail: "john@example.com",
				Channel:       "web",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
