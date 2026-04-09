package britive

import (
	"encoding/json"
	"net/http"
	"testing"
)

// ---------------------------------------------------------------------------
// catalogToCloud
// ---------------------------------------------------------------------------

func TestCatalogToCloud(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"AWS", "aws"},
		{"AWS_STANDALONE", "aws"},
		{"GCP", "gcp"},
		{"Azure", "azure"},
		{"unknown", "other"},
		{"", "other"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := catalogToCloud(tc.input)
			if got != tc.want {
				t.Errorf("catalogToCloud(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ListAccess
// ---------------------------------------------------------------------------

func TestListAccess_Success(t *testing.T) {
	payload := []AppAccess{
		{
			AppContainerID: "app-1",
			AppName:        "MyAWSApp",
			CatalogAppName: "AWS",
			Profiles: []PAP{
				{
					ProfileID:   "prof-1",
					ProfileName: "ReadOnly",
					PapID:       101,
					Environments: []Env{
						{EnvironmentID: "env-1", EnvironmentName: "dev"},
						{EnvironmentID: "env-2", EnvironmentName: "prod"},
					},
				},
			},
		},
		{
			AppContainerID: "app-2",
			AppName:        "MyGCPApp",
			CatalogAppName: "GCP",
			Profiles: []PAP{
				{
					ProfileID:   "prof-2",
					ProfileName: "Admin",
					PapID:       202,
					Environments: []Env{
						{EnvironmentID: "env-3", EnvironmentName: "staging"},
					},
				},
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/access" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})

	c := newTestClient(t, handler)
	entries, err := c.ListAccess()
	if err != nil {
		t.Fatalf("ListAccess() returned unexpected error: %v", err)
	}

	// app-1 has 2 environments, app-2 has 1 → 3 total entries
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Spot-check first entry (app-1, ReadOnly, dev)
	e0 := entries[0]
	if e0.AppName != "MyAWSApp" {
		t.Errorf("entries[0].AppName = %q; want %q", e0.AppName, "MyAWSApp")
	}
	if e0.ProfileName != "ReadOnly" {
		t.Errorf("entries[0].ProfileName = %q; want %q", e0.ProfileName, "ReadOnly")
	}
	if e0.ProfileID != "prof-1" {
		t.Errorf("entries[0].ProfileID = %q; want %q", e0.ProfileID, "prof-1")
	}
	if e0.EnvironmentName != "dev" {
		t.Errorf("entries[0].EnvironmentName = %q; want %q", e0.EnvironmentName, "dev")
	}
	if e0.EnvironmentID != "env-1" {
		t.Errorf("entries[0].EnvironmentID = %q; want %q", e0.EnvironmentID, "env-1")
	}
	if e0.Cloud != "aws" {
		t.Errorf("entries[0].Cloud = %q; want %q", e0.Cloud, "aws")
	}

	// Spot-check last entry (app-2, Admin, staging)
	e2 := entries[2]
	if e2.AppName != "MyGCPApp" {
		t.Errorf("entries[2].AppName = %q; want %q", e2.AppName, "MyGCPApp")
	}
	if e2.Cloud != "gcp" {
		t.Errorf("entries[2].Cloud = %q; want %q", e2.Cloud, "gcp")
	}
	if e2.EnvironmentName != "staging" {
		t.Errorf("entries[2].EnvironmentName = %q; want %q", e2.EnvironmentName, "staging")
	}
}

func TestListAccess_Empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	})

	c := newTestClient(t, handler)
	entries, err := c.ListAccess()
	if err != nil {
		t.Fatalf("ListAccess() returned unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListAccess_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})

	c := newTestClient(t, handler)
	_, err := c.ListAccess()
	if err == nil {
		t.Fatal("expected error from ListAccess() on 500 response, got nil")
	}
}
