package mackerel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestFindInvitation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/invitations" {
			t.Error("request URL should be but: ", req.URL.Path)
		}
		if req.Method != "GET" {
			t.Error("request method should be GET but: ", req.Method)
		}

		respJSON, _ := json.Marshal(map[string][]map[string]interface{}{
			"invitations": {
				{
					"email":     "test@example.com",
					"authority": "viewer",
					"expiresAt": 1560000000,
				},
			},
		})
		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	client, _ := NewClientWithOptions("dummy-key", ts.URL, false)
	invitations, err := client.FindInvitations()

	if err != nil {
		t.Error("err should be nil but: ", err)
	}

	if invitations[0].Email != "test@example.com" {
		t.Error("request sends json including email but: ", invitations[0].Email)
	}

	if invitations[0].Authority != "viewer" {
		t.Error("request sends json including authority but: ", invitations[0].Authority)
	}

	if invitations[0].ExpiresAt != 1560000000 {
		t.Error("request sends json including joinedAt but: ", invitations[0].ExpiresAt)
	}
}

func TestCreateInvitation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/invitations" {
			t.Error("request URL should be but: ", req.URL.Path)
		}
		if req.Method != http.MethodPost {
			t.Error("request method should be POST but: ", req.Method)
		}

		body, _ := io.ReadAll(req.Body)

		var invitation Invitation
		if err := json.Unmarshal(body, &invitation); err != nil {
			t.Fatal("request body should be decoded as json", string(body))
		}

		respJSON, _ := json.Marshal(map[string]interface{}{
			"email":     "test@example.com",
			"authority": "viewer",
			"expiresAt": 1560000000,
		})
		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	client, _ := NewClientWithOptions("dummy-key", ts.URL, false)

	param := &Invitation{
		Email:     "test@example.com",
		Authority: "viewer",
	}

	got, err := client.CreateInvitation(param)
	if err != nil {
		t.Error("err should be nil but: ", err)
	}

	want := &Invitation{
		Email:     "test@example.com",
		Authority: "viewer",
		ExpiresAt: 1560000000,
	}

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("fail to get correct data: diff (-got +want)\n%s", diff)
	}
}
