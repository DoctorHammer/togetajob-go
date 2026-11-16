package togetajob

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewClient(WithBaseURL(srv.URL))
	return c, srv.Close
}

func TestHealth(t *testing.T) {
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true,"lastRefresh":"2026-11-16"}`))
	})
	defer close()

	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !h.OK || h.LastRefresh != "2026-11-16" {
		t.Fatalf("unexpected health: %+v", h)
	}
}

func TestExams(t *testing.T) {
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"about":"x","license":"CC-BY","checked":"2026-11-16","source":"DCAS","exams":[{"number":"7131","title":"Emergency Medical Specialist - EMT","titleCode":null,"filingOpens":"2026-10-07","filingCloses":"2026-11-18","closeStatus":"confirmed","confirmedBy":"DCAS","fee":"$47","feeWaivers":"Fee waivers available","testExpected":"2027-01","expectedTestTakers":"","notice":"","page":"/exams/7131"}]}`))
	})
	defer close()

	f, err := c.Exams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Exams) != 1 || f.Exams[0].Number != "7131" {
		t.Fatalf("unexpected exams: %+v", f.Exams)
	}
	if f.Exams[0].TitleCode != nil {
		t.Fatalf("expected null titleCode, got %v", *f.Exams[0].TitleCode)
	}
}

func TestGuides(t *testing.T) {
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"about":"x","license":"CC-BY","guides":[{"id":"nyc-911-operator","title":"911 Operator","officialTitle":"Police Communications Technician","agency":"NYPD Communications Section","summary":"Answer 911 calls.","startingPay":"$51,790/yr","startingPayAnnual":51790,"payNote":"","paySource":"DCAS","examRequired":true,"nextFiling":"","timeToHired":"6 months","stepsWrittenFor":"","steps":[{"title":"Apply","text":"File in OASys."}],"sources":[],"page":"/guides/nyc-911-operator"}]}`))
	})
	defer close()

	f, err := c.Guides(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Guides) != 1 || f.Guides[0].ID != "nyc-911-operator" {
		t.Fatalf("unexpected guides: %+v", f.Guides)
	}
	if f.Guides[0].ExamRequired == nil || !*f.Guides[0].ExamRequired {
		t.Fatalf("expected examRequired true")
	}
}

func TestJobsDecodesPositionalRows(t *testing.T) {
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"asOf":"2026-11-16","categories":["Office"],"rows":[["chief-financial-officer-808149","Chief Financial Officer","Department of Environmental Protection","$250,000 to $282,253 a year",250000,"other",[5,10,13],["office-admin","technology"],"Queens",1,0,"95202","2026-12-27","2026-10-28",1,"808149",1,"dep tech data fin"]],"guides":[]}`))
	})
	defer close()

	f, err := c.Jobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(f.Rows))
	}
	j := f.Rows[0]
	if j.Slug != "chief-financial-officer-808149" || j.Title != "Chief Financial Officer" {
		t.Fatalf("unexpected job identity: %+v", j)
	}
	if j.PaySortKey != 250000 || j.Tier != "other" || j.BoroughLabel != "Queens" {
		t.Fatalf("unexpected job fields: %+v", j)
	}
	if !j.FullTime || j.ExamRequired == nil || *j.ExamRequired {
		t.Fatalf("expected full-time and no exam: %+v", j)
	}
	if j.JobID != "808149" || j.PayFrequency != "annual" || j.ClosesAt != "2026-12-27" {
		t.Fatalf("unexpected job details: %+v", j)
	}
	if j.Positions == nil || *j.Positions != 1 {
		t.Fatalf("expected 1 position: %+v", j)
	}
}

func TestPosting(t *testing.T) {
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "808867" {
			t.Fatalf("unexpected id %q", r.URL.Query().Get("id"))
		}
		_, _ = w.Write([]byte(`{"found":true,"jobId":"808867","slug":"space-planner-technology-specialist-808867","title":"Space Planner Technology Specialist","civilTitle":"IT Project Specialist","agency":"Dept of Citywide Admin Svcs","titleCode":"06797","lastSeen":"2026-11-16","stillListed":true,"trackedSince":"2026-09-23","timesPosted":8,"openNow":6,"officialUrl":"https://cityjobs.nyc.gov/job/808867"}`))
	})
	defer close()

	p, err := c.Posting(context.Background(), "808867")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Space Planner Technology Specialist" || !p.StillListed {
		t.Fatalf("unexpected posting: %+v", p)
	}
}

func TestPostingNotFound(t *testing.T) {
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"found":false,"jobId":"000000"}`))
	})
	defer close()

	_, err := c.Posting(context.Background(), "000000")
	if err == nil {
		t.Fatal("expected an error for a missing posting")
	}
	if ae, ok := err.(*APIError); !ok || ae.Status != http.StatusNotFound {
		t.Fatalf("expected a 404 APIError, got %T %v", err, err)
	}
}

func TestCreateAndRevokeKey(t *testing.T) {
	var seenKey string
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/keys" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"key":"tj_live_abc","limit":600,"window_seconds":60,"expires_at":"2027-11-16T00:00:00Z","note":"Keep this key secret."}`))
		case r.URL.Path == "/api/keys" && r.Method == http.MethodDelete:
			seenKey = r.Header.Get("X-Api-Key")
			_, _ = w.Write([]byte(`{"revoked":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	defer close()

	k, err := c.CreateKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if k.Key != "tj_live_abc" || k.Limit != 600 {
		t.Fatalf("unexpected key: %+v", k)
	}
	if err := c.RevokeKey(context.Background(), "tj_live_abc"); err != nil {
		t.Fatal(err)
	}
	if seenKey != "tj_live_abc" {
		t.Fatalf("revoke sent the wrong key: %q", seenKey)
	}
}

func TestAPIKeySentAsHeader(t *testing.T) {
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "tj_live_xyz" {
			t.Fatalf("expected X-Api-Key header, got %q", r.Header.Get("X-Api-Key"))
		}
		_, _ = w.Write([]byte(`{"ok":true,"lastRefresh":"2026-11-16"}`))
	})
	defer close()

	keyed := NewClient(WithBaseURL(c.baseURL), WithAPIKey("tj_live_xyz"))
	if _, err := keyed.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestAPIError(t *testing.T) {
	c, close := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"Rate limit exceeded","code":"rate_limited","resolution":"Wait a minute, or create a free key."}`))
	})
	defer close()

	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	ae, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if ae.Status != http.StatusTooManyRequests || ae.Code != "rate_limited" {
		t.Fatalf("unexpected APIError: %+v", ae)
	}
	if got := ae.Error(); got == "" {
		t.Fatal("empty error string")
	}
}

// TestLiveHealth is a smoke test against the real site. It is skipped unless
// -run=TestLive is passed, so `go test ./...` stays offline and fast.
func TestLiveHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	c := NewClient()
	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !h.OK {
		t.Fatalf("live health not ok: %+v", h)
	}
	b, _ := json.Marshal(h)
	t.Logf("live health: %s", b)
}
