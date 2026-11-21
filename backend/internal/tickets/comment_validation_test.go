package tickets

import "testing"

func TestCreateCommentRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request CreateCommentRequest
		wantErr bool
	}{
		{
			name: "valid agent comment",
			request: CreateCommentRequest{
				AuthorType: "agent",
				Body:       "I am investigating this issue.",
			},
		},
		{
			name: "valid customer comment",
			request: CreateCommentRequest{
				AuthorType: "customer",
				Body:       "I need help with my payment.",
			},
		},
		{
			name: "missing author type",
			request: CreateCommentRequest{
				Body: "Hello",
			},
			wantErr: true,
		},
		{
			name: "invalid author type",
			request: CreateCommentRequest{
				AuthorType: "admin",
				Body:       "Hello",
			},
			wantErr: true,
		},
		{
			name: "empty body",
			request: CreateCommentRequest{
				AuthorType: "agent",
				Body:       "   ",
			},
			wantErr: true,
		},
		{
			name: "body exceeds maximum length",
			request: CreateCommentRequest{
				AuthorType: "agent",
				Body:       string(make([]byte, 10001)),
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