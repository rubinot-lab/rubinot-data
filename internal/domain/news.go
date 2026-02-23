package domain

type NewsResult struct {
	ID          int    `json:"id"`
	Date        string `json:"date,omitempty"`
	Title       string `json:"title,omitempty"`
	Category    string `json:"category,omitempty"`
	Type        string `json:"type"`
	Content     string `json:"content,omitempty"`
	ContentHTML string `json:"content_html,omitempty"`
}

type NewsListEntry struct {
	ID       int    `json:"id,omitempty"`
	Date     string `json:"date,omitempty"`
	Title    string `json:"title,omitempty"`
	Category string `json:"category,omitempty"`
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
}

type NewsListResult struct {
	Mode        string          `json:"mode"`
	ArchiveDays int             `json:"archive_days,omitempty"`
	Entries     []NewsListEntry `json:"entries"`
}
