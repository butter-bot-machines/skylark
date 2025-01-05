package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockProvider struct {
	response *Response
	err      error
	delay    time.Duration
}

func (m *mockProvider) Send(ctx context.Context, prompt string, opts *RequestOptions) (*Response, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}

	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockProvider) Close() error {
	return nil
}

func TestProviderInterface(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		prompt   string
		want     *Response
		wantErr  error
	}{
		{
			name: "successful response",
			provider: &mockProvider{
				response: &Response{
					Content: "test response",
					Usage: Usage{
						PromptTokens:     10,
						CompletionTokens: 20,
						TotalTokens:      30,
					},
				},
			},
			prompt: "test prompt",
			want: &Response{
				Content: "test response",
				Usage: Usage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			},
		},
		{
			name: "error response",
			provider: &mockProvider{
				err: &Error{
					Code:    ErrRateLimit,
					Message: "rate limit exceeded",
				},
			},
			prompt:  "test prompt",
			wantErr: &Error{Code: ErrRateLimit, Message: "rate limit exceeded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := tt.provider.Send(ctx, tt.prompt, DefaultRequestOptions)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Content != tt.want.Content {
				t.Errorf("expected content %q, got %q", tt.want.Content, got.Content)
			}

			if got.Usage != tt.want.Usage {
				t.Errorf("expected usage %+v, got %+v", tt.want.Usage, got.Usage)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	provider := &mockProvider{
		response: &Response{Content: "test"},
		delay:    100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := provider.Send(ctx, "test", DefaultRequestOptions)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestContextTimeout(t *testing.T) {
	provider := &mockProvider{
		response: &Response{Content: "test"},
		delay:    100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	_, err := provider.Send(ctx, "test", DefaultRequestOptions)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded error, got %v", err)
	}
}
