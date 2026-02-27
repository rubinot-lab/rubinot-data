package domain

type NewsCategory struct {
	ID      int    `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Slug    string `json:"slug,omitempty"`
	Color   string `json:"color,omitempty"`
	Icon    string `json:"icon,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type NewsResult struct {
	ID          int          `json:"id"`
	Date        string       `json:"date,omitempty"`
	Title       string       `json:"title,omitempty"`
	Category    string       `json:"category,omitempty"`
	CategoryRef NewsCategory `json:"category_ref,omitempty"`
	Type        string       `json:"type"`
	Content     string       `json:"content,omitempty"`
	ContentHTML string       `json:"content_html,omitempty"`
	Author      string       `json:"author,omitempty"`
	Slug        string       `json:"slug,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	CoverImage  string       `json:"cover_image,omitempty"`
}

type NewsListEntry struct {
	ID          int          `json:"id,omitempty"`
	Date        string       `json:"date,omitempty"`
	Title       string       `json:"title,omitempty"`
	Category    string       `json:"category,omitempty"`
	Type        string       `json:"type"`
	URL         string       `json:"url,omitempty"`
	Author      string       `json:"author,omitempty"`
	Slug        string       `json:"slug,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	Message     string       `json:"message,omitempty"`
	CategoryRef NewsCategory `json:"category_ref,omitempty"`
}

type NewsListResult struct {
	Mode        string          `json:"mode"`
	ArchiveDays int             `json:"archive_days,omitempty"`
	Entries     []NewsListEntry `json:"entries"`
}
