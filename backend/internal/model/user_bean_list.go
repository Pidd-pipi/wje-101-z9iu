package model

import "time"

// UserBeanList is a user's "want to drink" entry for one coffee bean.
// One row per (user, bean). A bean with at least one matching tasting note
// is presented as "drunk"; when the note count drops back to zero the same
// row keeps the bean in the "want to drink" group.
type UserBeanList struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;uniqueIndex:uni_user_bean_list,priority:1" json:"user_id"`
	BeanID    uint       `gorm:"not null;uniqueIndex:uni_user_bean_list,priority:2" json:"bean_id"`
	User      User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Bean      CoffeeBean `gorm:"foreignKey:BeanID;constraint:OnDelete:CASCADE" json:"-"`
	CreatedAt time.Time  `json:"created_at"`
}
