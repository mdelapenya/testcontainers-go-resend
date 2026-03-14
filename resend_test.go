package resend_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/mdelapenya/testcontainers-go-resend"
)

// mockID is the UUID used in the enriched OpenAPI spec examples.
// Microcks dispatches path-parameter endpoints by matching the parameter
// value to the example value, so tests must use this exact ID.
const mockID = "479e3145-dd38-476b-932c-529ceb705947"

// doRequest performs an HTTP request and returns the status code and raw body.
func doRequest(t *testing.T, method, url, body string) (int, []byte) {
	t.Helper()

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, respBody
}

// doJSONRequest performs an HTTP request, asserts the expected status code,
// and decodes the response as JSON.
func doJSONRequest(t *testing.T, method, url, body string, expectedStatus int) map[string]any {
	t.Helper()

	status, respBody := doRequest(t, method, url, body)
	assert.Equalf(t, expectedStatus, status, "unexpected status code; body: %s", string(respBody))

	if len(respBody) == 0 {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Logf("response is not JSON (status %d): %s", status, string(respBody))
		return nil
	}
	return result
}

func TestResend(t *testing.T) {
	ctx := context.Background()

	ctr, err := resend.Run(ctx, resend.DefaultImage)
	testcontainers.CleanupContainer(t, ctr)
	require.NoError(t, err)

	baseURL, err := ctr.BaseURL(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, baseURL)

	// Emails API
	t.Run("emails", func(t *testing.T) {
		t.Run("send email", func(t *testing.T) {
			body := `{
				"from": "Acme <onboarding@resend.dev>",
				"to": ["delivered@resend.dev"],
				"subject": "Hello World",
				"html": "<p>Congrats on sending your first email!</p>"
			}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/emails", body, http.StatusOK)
			require.Contains(t, result, "id")
		})

		t.Run("list emails", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/emails", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve email", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/emails/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("update email", func(t *testing.T) {
			body := `{"scheduled_at": "2024-01-01T00:00:00Z"}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/emails/"+mockID, body, http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("cancel email", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodPost, baseURL+"/emails/"+mockID+"/cancel", "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("send batch emails", func(t *testing.T) {
			body := `[{
				"from": "Acme <onboarding@resend.dev>",
				"to": ["delivered@resend.dev"],
				"subject": "Hello 1",
				"html": "<p>Email 1</p>"
			}]`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/emails/batch", body, http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("list attachments", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/emails/"+mockID+"/attachments", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve attachment", func(t *testing.T) {
			t.Skip("Microcks' URI_PARTS dispatcher constructs a compound dispatch key for multi-path-param endpoints that doesn't match named examples")
			result := doJSONRequest(t, http.MethodGet, baseURL+"/emails/"+mockID+"/attachments/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Receiving Emails API
	t.Run("receiving emails", func(t *testing.T) {
		t.Run("list received emails", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/emails/receiving", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve received email", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/emails/receiving/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Domains API
	t.Run("domains", func(t *testing.T) {
		t.Run("create domain", func(t *testing.T) {
			body := `{"name": "example.com"}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/domains", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list domains", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/domains", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve domain", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/domains/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("update domain", func(t *testing.T) {
			body := `{"open_tracking": true}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/domains/"+mockID, body, http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete domain", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/domains/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("verify domain", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodPost, baseURL+"/domains/"+mockID+"/verify", "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// API Keys
	t.Run("api-keys", func(t *testing.T) {
		t.Run("create api key", func(t *testing.T) {
			body := `{"name": "test-key"}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/api-keys", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list api keys", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/api-keys", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("delete api key", func(t *testing.T) {
			t.Skip("Resend spec defines no response body for DELETE /api-keys/{api_key_id}, so Microcks cannot mock it")
			status, _ := doRequest(t, http.MethodDelete, baseURL+"/api-keys/"+mockID, "")
			require.Equal(t, http.StatusOK, status)
		})
	})

	// Templates API
	t.Run("templates", func(t *testing.T) {
		t.Run("create template", func(t *testing.T) {
			body := `{
				"name": "Monthly Newsletter",
				"subject": "Newsletter {{month}}",
				"html": "<h1>Hello {{name}}</h1>"
			}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/templates", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list templates", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/templates", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve template", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/templates/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("update template", func(t *testing.T) {
			body := `{"name": "Updated Newsletter"}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/templates/"+mockID, body, http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete template", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/templates/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("publish template", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodPost, baseURL+"/templates/"+mockID+"/publish", "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("duplicate template", func(t *testing.T) {
			body := `{"name": "Duplicated Newsletter"}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/templates/"+mockID+"/duplicate", body, http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Audiences API (deprecated)
	t.Run("audiences", func(t *testing.T) {
		t.Run("create audience", func(t *testing.T) {
			body := `{"name": "Registered Users"}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/audiences", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list audiences", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/audiences", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve audience", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/audiences/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete audience", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/audiences/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Contacts API
	t.Run("contacts", func(t *testing.T) {
		t.Run("create contact", func(t *testing.T) {
			body := `{
				"email": "user@example.com",
				"first_name": "John",
				"last_name": "Doe"
			}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/contacts", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list contacts", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/contacts", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve contact", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/contacts/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("update contact", func(t *testing.T) {
			body := `{"first_name": "Jane"}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/contacts/"+mockID, body, http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete contact", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/contacts/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Broadcasts API
	t.Run("broadcasts", func(t *testing.T) {
		t.Run("create broadcast", func(t *testing.T) {
			body := `{
				"name": "Product Launch",
				"from": "Acme <news@resend.dev>",
				"subject": "New Product!"
			}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/broadcasts", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list broadcasts", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/broadcasts", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve broadcast", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/broadcasts/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("update broadcast", func(t *testing.T) {
			body := `{"name": "Updated Product Launch"}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/broadcasts/"+mockID, body, http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete broadcast", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/broadcasts/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("send broadcast", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodPost, baseURL+"/broadcasts/"+mockID+"/send", "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Webhooks API
	t.Run("webhooks", func(t *testing.T) {
		t.Run("create webhook", func(t *testing.T) {
			body := `{
				"endpoint": "https://webhook.example.com/handler",
				"events": ["email.sent", "email.delivered"]
			}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/webhooks", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list webhooks", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/webhooks", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve webhook", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/webhooks/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("update webhook", func(t *testing.T) {
			body := `{
				"endpoint": "https://webhook.example.com/new-handler",
				"events": ["email.sent"]
			}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/webhooks/"+mockID, body, http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete webhook", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/webhooks/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Segments API
	t.Run("segments", func(t *testing.T) {
		t.Run("create segment", func(t *testing.T) {
			body := `{"name": "Active Users"}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/segments", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list segments", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/segments", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve segment", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/segments/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete segment", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/segments/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Topics API
	t.Run("topics", func(t *testing.T) {
		t.Run("create topic", func(t *testing.T) {
			body := `{"name": "Product Updates"}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/topics", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list topics", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/topics", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve topic", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/topics/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("update topic", func(t *testing.T) {
			body := `{"name": "Updated Product Updates"}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/topics/"+mockID, body, http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete topic", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/topics/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Contact Properties API
	t.Run("contact-properties", func(t *testing.T) {
		t.Run("create contact property", func(t *testing.T) {
			body := `{"name": "Company", "type": "string"}`
			result := doJSONRequest(t, http.MethodPost, baseURL+"/contact-properties", body, http.StatusCreated)
			require.Contains(t, result, "id")
		})

		t.Run("list contact properties", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/contact-properties", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("retrieve contact property", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/contact-properties/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("update contact property", func(t *testing.T) {
			body := `{"name": "Updated Company"}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/contact-properties/"+mockID, body, http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("delete contact property", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/contact-properties/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	// Contact Segments & Topics management
	t.Run("contact-segments", func(t *testing.T) {
		t.Run("list contact segments", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/contacts/"+mockID+"/segments", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("add contact to segment", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodPost, baseURL+"/contacts/"+mockID+"/segments/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})

		t.Run("remove contact from segment", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodDelete, baseURL+"/contacts/"+mockID+"/segments/"+mockID, "", http.StatusOK)
			require.NotNil(t, result)
		})
	})

	t.Run("contact-topics", func(t *testing.T) {
		t.Run("list contact topics", func(t *testing.T) {
			result := doJSONRequest(t, http.MethodGet, baseURL+"/contacts/"+mockID+"/topics", "", http.StatusOK)
			require.Contains(t, result, "data")
		})

		t.Run("update contact topics", func(t *testing.T) {
			body := `{
				"topics": [
					{"id": "` + mockID + `", "subscription": "opt_in"}
				]
			}`
			result := doJSONRequest(t, http.MethodPatch, baseURL+"/contacts/"+mockID+"/topics", body, http.StatusOK)
			require.NotNil(t, result)
		})
	})
}
