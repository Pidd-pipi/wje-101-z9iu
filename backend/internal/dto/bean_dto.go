package dto

import "github.com/wjecoffeetaste/wjecoffeetaste/internal/model"

// BeanCreateRequest creates/updates a coffee bean.
type BeanCreateRequest struct {
	Name          string `json:"name" binding:"required,max=128"`
	Origin        string `json:"origin" binding:"omitempty,max=128"`
	ProcessMethod string `json:"process_method" binding:"required"`
	FlavorTags    string `json:"flavor_tags"`
	Description   string `json:"description"`
}

// BeanListItem is a bean card enriched with the public tasting-note tally and,
// for an authenticated viewer, the viewer's personal list state.
type BeanListItem struct {
	model.CoffeeBean
	NoteCount   int64 `json:"note_count"`
	MyNoteCount int64 `json:"my_note_count"`
	InList      bool  `json:"in_list"`
}

// UserBeanGroupData is the personal profile split of a user's beans.
type UserBeanGroupData struct {
	Want  []BeanListItem `json:"want"`
	Drunk []BeanListItem `json:"drunk"`
}
