package models

import "time"

// User is a Threads profile summary.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name,omitempty"`
	Bio      string `json:"bio,omitempty"`
	Avatar   string `json:"avatar_url,omitempty"`
	Verified bool   `json:"verified,omitempty"`
	Followers int   `json:"followers_count,omitempty"`
	Following int   `json:"following_count,omitempty"`
	Threads   int   `json:"threads_count,omitempty"`
}

// Post is a Threads post / media object.
type Post struct {
	ID           string    `json:"id"`
	Text         string    `json:"text"`
	Username     string    `json:"username"`
	UserID       string    `json:"user_id,omitempty"`
	Permalink    string    `json:"permalink,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	LikeCount    int       `json:"like_count,omitempty"`
	ReplyCount   int       `json:"reply_count,omitempty"`
	RepostCount  int       `json:"repost_count,omitempty"`
	MediaType    string    `json:"media_type,omitempty"`
	MediaURL     string    `json:"media_url,omitempty"`
	IsReply      bool      `json:"is_reply,omitempty"`
	ReplyToID    string    `json:"reply_to_id,omitempty"`
	TopicTag     string    `json:"topic_tag,omitempty"`
	LikedByMe    bool      `json:"liked_by_me,omitempty"`
}

// Thread is a root post plus its replies.
type Thread struct {
	Root    Post   `json:"root"`
	Replies []Post `json:"replies"`
}

// FeedPage is a paginated list of posts.
type FeedPage struct {
	Posts      []Post `json:"posts"`
	NextCursor string `json:"next_cursor,omitempty"`
	// Source: home | following | discover | search | empty
	Source string `json:"source,omitempty"`
	// Hint explains empty / fallback feeds (shown in TUI/CLI).
	Hint string `json:"hint,omitempty"`
}

// PublishResult is returned after creating a post.
type PublishResult struct {
	ID        string `json:"id"`
	Permalink string `json:"permalink,omitempty"`
	Container string `json:"container_id,omitempty"`
}

// AuthStatus describes the current auth mode.
type AuthStatus struct {
	Mode     string `json:"mode"` // demo | token | oauth
	Username string `json:"username,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Ready    bool   `json:"ready"`
}
