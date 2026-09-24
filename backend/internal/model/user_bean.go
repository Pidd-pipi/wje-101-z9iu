package model

import "time"

// UserBean records a bean placed in a user's to-drink list.
// The row is created when a user adds the bean to the list (or publishes
// a tasting note on it); it is kept while the bean is 喝过 and is only
// removed when the user explicitly moves it out of 待喝 with zero notes.
type UserBean struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_user_bean,unique;not null" json:"user_id"`
	BeanID    uint      `gorm:"index:idx_user_bean,unique;not null" json:"bean_id"`
	CreatedAt time.Time `json:"created_at"`
}
