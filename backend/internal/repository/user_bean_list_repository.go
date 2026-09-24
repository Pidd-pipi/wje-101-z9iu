package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
)

// UserBeanListRepository handles user bean list ("want to drink") persistence.
type UserBeanListRepository struct{ db *gorm.DB }

// NewUserBeanListRepository creates the repository.
func NewUserBeanListRepository(db *gorm.DB) *UserBeanListRepository {
	return &UserBeanListRepository{db: db}
}

// Add inserts a user-bean entry. It is idempotent: a concurrent or repeated
// insert that hits the unique (user_id, bean_id) index is treated as success.
func (r *UserBeanListRepository) Add(userID, beanID uint) error {
	err := translate(r.db.Create(&model.UserBeanList{UserID: userID, BeanID: beanID}).Error)
	if errors.Is(err, ErrDuplicate) {
		return nil
	}
	return err
}

// Delete removes a user-bean entry (move out of the want-to-drink list).
func (r *UserBeanListRepository) Delete(userID, beanID uint) error {
	res := r.db.Where("user_id = ? AND bean_id = ?", userID, beanID).
		Delete(&model.UserBeanList{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListBeanIDsByUser returns the bean ids tracked by a user.
func (r *UserBeanListRepository) ListBeanIDsByUser(userID uint) ([]uint, error) {
	ids := make([]uint, 0)
	if err := r.db.Model(&model.UserBeanList{}).
		Where("user_id = ?", userID).
		Order("id DESC").
		Pluck("bean_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// DeleteByBean removes all entries of a bean.
func (r *UserBeanListRepository) DeleteByBean(beanID uint) error {
	return r.db.Where("bean_id = ?", beanID).Delete(&model.UserBeanList{}).Error
}
