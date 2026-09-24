package repository

import (
	"gorm.io/gorm"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
)

// UserBeanRepository handles per-user to-drink list persistence.
type UserBeanRepository struct{ db *gorm.DB }

// NewUserBeanRepository creates the repository.
func NewUserBeanRepository(db *gorm.DB) *UserBeanRepository { return &UserBeanRepository{db: db} }

// Create inserts a user-bean tracking row.
func (r *UserBeanRepository) Create(ub *model.UserBean) error {
	return translate(r.db.Create(ub).Error)
}

// Find locates a tracking row by user and bean.
func (r *UserBeanRepository) Find(userID, beanID uint) (*model.UserBean, error) {
	var ub model.UserBean
	if err := translate(r.db.Where("user_id = ? AND bean_id = ?", userID, beanID).First(&ub).Error); err != nil {
		return nil, err
	}
	return &ub, nil
}

// Delete removes a tracking row by user and bean.
func (r *UserBeanRepository) Delete(userID, beanID uint) error {
	res := r.db.Where("user_id = ? AND bean_id = ?", userID, beanID).Delete(&model.UserBean{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// BeanIDsByUser returns the bean ids tracked by a user, oldest first.
func (r *UserBeanRepository) BeanIDsByUser(userID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&model.UserBean{}).
		Where("user_id = ?", userID).
		Order("id ASC").
		Pluck("bean_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
