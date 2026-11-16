package togetajob

import "encoding/json"

// Health is the /api/health response.
type Health struct {
	OK          bool   `json:"ok"`
	LastRefresh string `json:"lastRefresh"`
}

// APIKey is the /api/keys response. Key is returned once, in full, and only
// its hash is stored server-side. Limit is requests per minute, WindowSeconds
// the window those requests are counted over, ExpiresAt the key's expiry
// (a year after creation).
type APIKey struct {
	Key           string `json:"key"`
	Limit         int    `json:"limit"`
	WindowSeconds int    `json:"window_seconds"`
	ExpiresAt     string `json:"expires_at"`
	Note          string `json:"note"`
}

// Exam is one row of the civil service exam calendar.
type Exam struct {
	Number             string  `json:"number"`
	Title              string  `json:"title"`
	TitleCode          *string `json:"titleCode"`
	FilingOpens        string  `json:"filingOpens"`
	FilingCloses       string  `json:"filingCloses"`
	CloseStatus        string  `json:"closeStatus"`
	ConfirmedBy        string  `json:"confirmedBy"`
	Fee                string  `json:"fee"`
	FeeWaivers         string  `json:"feeWaivers"`
	TestExpected       string  `json:"testExpected"`
	ExpectedTestTakers string  `json:"expectedTestTakers"`
	Notice             string  `json:"notice"`
	Page               string  `json:"page"`
}

// ExamsFeed is the /data/exams.json feed: the exam calendar plus its
// provenance (when it was checked, where the data came from).
type ExamsFeed struct {
	About   string `json:"about"`
	License string `json:"license"`
	Checked string `json:"checked"`
	Source  string `json:"source"`
	Exams   []Exam `json:"exams"`
}

// GuideStep is one step in a job guide.
type GuideStep struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Guide is one step-by-step job guide: what an entry-level city title pays,
// whether it needs an exam, and the exact steps to get hired.
type Guide struct {
	ID                string      `json:"id"`
	Title             string      `json:"title"`
	OfficialTitle     string      `json:"officialTitle"`
	Agency            string      `json:"agency"`
	Summary           string      `json:"summary"`
	StartingPay       string      `json:"startingPay"`
	StartingPayAnnual *int        `json:"startingPayAnnual"`
	PayNote           string      `json:"payNote"`
	PaySource         string      `json:"paySource"`
	ExamRequired      *bool       `json:"examRequired"`
	NextFiling        string      `json:"nextFiling"`
	TimeToHired       string      `json:"timeToHired"`
	StepsWrittenFor   string      `json:"stepsWrittenFor"`
	Steps             []GuideStep `json:"steps"`
	Sources           []string    `json:"sources"`
	Page              string      `json:"page"`
}

// GuidesFeed is the /data/guides.json feed.
type GuidesFeed struct {
	About   string  `json:"about"`
	License string  `json:"license"`
	Guides  []Guide `json:"guides"`
}

// Job is one open city job posting, decoded from the compact positional rows
// of /jobs/index.json (see fromIndexRow in the site's src/lib/jobrow.js).
type Job struct {
	Slug            string   `json:"slug"`
	Title           string   `json:"title"`
	Agency          string   `json:"agency"`
	SalaryText      string   `json:"salaryText"`
	PaySortKey      int      `json:"paySortKey"`
	Tier            string   `json:"tier"`
	CategoryIndexes []int    `json:"categoryIndexes"`
	Tags            []string `json:"tags"`
	BoroughLabel    string   `json:"boroughLabel"`
	FullTime        bool     `json:"fullTime"`
	ExamRequired    *bool    `json:"examRequired"`
	TitleCode       string   `json:"titleCode"`
	ClosesAt        string   `json:"closesAt"`
	PostedAt        string   `json:"postedAt"`
	Positions       *int     `json:"positions"`
	JobID           string   `json:"jobId"`
	PayFrequency    string   `json:"payFrequency"`
	Keywords        string   `json:"keywords"`
}

// jobRow is the compact positional row the index ships: [slug, title,
// agency, salaryText, paySortKey, tier, categoryIndexes, tags,
// boroughLabel, fullTime (1/0), examRequired (1/0/null), titleCode,
// closesAt, postedAt, positions, jobId, payFrequency (1=annual),
// keywords].
type jobRow [18]json.RawMessage

func (r jobRow) job() (Job, error) {
	var j Job
	str := func(i int, dst *string) error { return json.Unmarshal(r[i], dst) }
	num := func(i int, dst *int) error { return json.Unmarshal(r[i], dst) }
	if err := str(0, &j.Slug); err != nil {
		return j, err
	}
	if err := str(1, &j.Title); err != nil {
		return j, err
	}
	if err := str(2, &j.Agency); err != nil {
		return j, err
	}
	if err := str(3, &j.SalaryText); err != nil {
		return j, err
	}
	if err := num(4, &j.PaySortKey); err != nil {
		return j, err
	}
	if err := str(5, &j.Tier); err != nil {
		return j, err
	}
	if err := json.Unmarshal(r[6], &j.CategoryIndexes); err != nil {
		return j, err
	}
	if err := json.Unmarshal(r[7], &j.Tags); err != nil {
		return j, err
	}
	if err := str(8, &j.BoroughLabel); err != nil {
		return j, err
	}
	var fullTime int
	if err := num(9, &fullTime); err != nil {
		return j, err
	}
	j.FullTime = fullTime != 0
	var examRaw json.RawMessage
	if err := json.Unmarshal(r[10], &examRaw); err != nil {
		return j, err
	}
	if string(examRaw) != "null" {
		var v int
		if err := json.Unmarshal(examRaw, &v); err != nil {
			return j, err
		}
		b := v == 1
		j.ExamRequired = &b
	}
	if err := str(11, &j.TitleCode); err != nil {
		return j, err
	}
	if err := str(12, &j.ClosesAt); err != nil {
		return j, err
	}
	if err := str(13, &j.PostedAt); err != nil {
		return j, err
	}
	if string(r[14]) != "null" {
		var v int
		if err := json.Unmarshal(r[14], &v); err != nil {
			return j, err
		}
		j.Positions = &v
	}
	if err := str(15, &j.JobID); err != nil {
		return j, err
	}
	var freq int
	if err := num(16, &freq); err != nil {
		return j, err
	}
	if freq == 1 {
		j.PayFrequency = "annual"
	}
	if err := str(17, &j.Keywords); err != nil {
		return j, err
	}
	return j, nil
}

// JobGuide is one guide summary embedded in the jobs index: the guide's
// identity, its search text, and its exam window (if any).
type JobGuide struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Agency string `json:"agency"`
	Keys   string `json:"keys"`
	Exam   *struct {
		Number string `json:"n"`
		Start  string `json:"start"`
		End    string `json:"end"`
		Now    bool   `json:"now"`
		When   string `json:"when"`
	} `json:"exam"`
	NoExam bool `json:"noExam"`
}

// JobsIndex is the /jobs/index.json feed: every open posting, sorted easiest
// to get first, plus the category list and the date the data was checked.
type JobsIndex struct {
	AsOf       string     `json:"asOf"`
	Categories []string   `json:"categories"`
	Rows       []Job      `json:"-"`
	Guides     []JobGuide `json:"guides"`
}

// UnmarshalJSON decodes the compact positional rows into Jobs.
func (f *JobsIndex) UnmarshalJSON(data []byte) error {
	var raw struct {
		AsOf       string     `json:"asOf"`
		Categories []string   `json:"categories"`
		Rows       []jobRow   `json:"rows"`
		Guides     []JobGuide `json:"guides"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.AsOf = raw.AsOf
	f.Categories = raw.Categories
	f.Guides = raw.Guides
	f.Rows = make([]Job, 0, len(raw.Rows))
	for _, r := range raw.Rows {
		j, err := r.job()
		if err != nil {
			return err
		}
		f.Rows = append(f.Rows, j)
	}
	return nil
}

// Posting is the /api/posting response for one open posting.
type Posting struct {
	Found        bool   `json:"found"`
	JobID        string `json:"jobId"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	CivilTitle   string `json:"civilTitle"`
	Agency       string `json:"agency"`
	TitleCode    string `json:"titleCode"`
	LastSeen     string `json:"lastSeen"`
	StillListed  bool   `json:"stillListed"`
	TrackedSince string `json:"trackedSince"`
	TimesPosted  int    `json:"timesPosted"`
	OpenNow      int    `json:"openNow"`
	OfficialURL  string `json:"officialUrl"`
}
