package ds

import (
	"time"
)

type Commit struct {
	ID             string            `bson:"id"`
	ShortID        string            `bson:"short_id"`
	Title          string            `bson:"title"`
	AuthorName     string            `bson:"author_name"`
	AuthorEmail    string            `bson:"author_email"`
	AuthoredDate   *time.Time        `bson:"authored_date"`
	CommitterName  string            `bson:"committer_name"`
	CommitterEmail string            `bson:"committer_email"`
	CommittedDate  *time.Time        `bson:"committed_date"`
	CreatedAt      *time.Time        `bson:"created_at"`
	Message        string            `bson:"message"`
	ParentIDs      []string          `bson:"parent_ids"`
	ProjectID      int               `bson:"project_id"`
	Trailers       map[string]string `bson:"trailers"`
	WebURL         string            `bson:"web_url"`
	Stats          *CommitStats      `bson:"stats"`
}

type CommitStats struct {
	Additions int `bson:"additions"`
	Deletions int `bson:"deletions"`
	Total     int `bson:"total"`
}

// IsEqual checks if two merge requests are equal (according to basic information)
func (a *Commit) IsEqual(b *Commit) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.ID != b.ID {
		return false
	}

	if a.ShortID != b.ShortID {
		return false
	}

	if a.ProjectID != b.ProjectID {
		return false
	}

	if a.AuthorName != b.AuthorName {
		return false
	}

	if a.AuthorEmail != b.AuthorEmail {
		return false
	}

	if a.Title != b.Title {
		return false
	}

	if a.CommitterName != b.CommitterName {
		return false
	}

	if a.CommitterEmail != b.CommitterEmail {
		return false
	}

	if a.Message != b.Message {
		return false
	}

	if a.WebURL != b.WebURL {
		return false
	}

	if (a.AuthoredDate != nil && b.AuthoredDate != nil && !a.AuthoredDate.Equal(*b.AuthoredDate)) || (a.AuthoredDate == nil && b.AuthoredDate != nil) || (a.AuthoredDate != nil && b.AuthoredDate == nil) {
		return false
	}

	if (a.CreatedAt != nil && b.CreatedAt != nil && !a.CreatedAt.Equal(*b.CreatedAt)) || (a.CreatedAt == nil && b.CreatedAt != nil) || (a.CreatedAt != nil && b.CreatedAt == nil) {
		return false
	}

	return true
}
