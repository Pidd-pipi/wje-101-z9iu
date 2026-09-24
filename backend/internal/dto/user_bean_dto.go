package dto

import "github.com/wjecoffeetaste/wjecoffeetaste/internal/model"

// BeanListItem decorates a bean with public drink count and, when the
// request is authenticated, the current viewer's tracking state.
type BeanListItem struct {
	model.CoffeeBean
	NoteTotal     int64  `json:"note_total"`
	Tracked       bool   `json:"tracked"`
	TrackStatus   string `json:"track_status"`
	UserNoteCount int64  `json:"user_note_count"`
}

// UserBeanGroupItem is one bean in a profile want/tasted group.
type UserBeanGroupItem struct {
	model.CoffeeBean
	NoteCount int64               `json:"note_count"`
	Notes     []model.TastingNote `json:"notes"`
}

// UserBeanGroups splits a user's tracked beans into 待喝 and 喝过.
type UserBeanGroups struct {
	Want   []UserBeanGroupItem `json:"want"`
	Tasted []UserBeanGroupItem `json:"tasted"`
}
