package repository

import (
	"gorm.io/gorm"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
)

// TastingNoteRepository handles note persistence.
type TastingNoteRepository struct{ db *gorm.DB }

// NewTastingNoteRepository creates the repository.
func NewTastingNoteRepository(db *gorm.DB) *TastingNoteRepository {
	return &TastingNoteRepository{db: db}
}

// Create inserts a note.
func (r *TastingNoteRepository) Create(n *model.TastingNote) error {
	return translate(r.db.Create(n).Error)
}

// FindByID locates a note by id.
func (r *TastingNoteRepository) FindByID(id uint) (*model.TastingNote, error) {
	var n model.TastingNote
	if err := translate(r.db.First(&n, id).Error); err != nil {
		return nil, err
	}
	return &n, nil
}

// Update persists a note.
func (r *TastingNoteRepository) Update(n *model.TastingNote) error {
	return translate(r.db.Save(n).Error)
}

// Delete removes a note by id.
func (r *TastingNoteRepository) Delete(id uint) error {
	res := r.db.Delete(&model.TastingNote{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// List filters notes by roast/origin/keyword, ordered by like count or recency.
func (r *TastingNoteRepository) List(roast, origin, keyword string, hot bool, page, pageSize int) ([]model.TastingNote, int64, error) {
	var items []model.TastingNote
	var total int64
	q := r.db.Model(&model.TastingNote{})
	if roast != "" {
		q = q.Where("roast_level = ?", roast)
	}
	if origin != "" {
		q = q.Where("origin = ?", origin)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("coffee_name LIKE ? OR notes_text LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "id DESC"
	if hot {
		order = "overall_score DESC, id DESC"
	}
	if err := q.Order(order).Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListByUser returns notes of a user.
func (r *TastingNoteRepository) ListByUser(userID uint) ([]model.TastingNote, error) {
	var items []model.TastingNote
	if err := r.db.Where("user_id = ?", userID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// BeanNameCount is the number of notes a user wrote for one coffee name.
type BeanNameCount struct {
	CoffeeName string
	Count      int64
}

// CountGroupByBeanName returns note counts grouped by coffee_name for a user.
func (r *TastingNoteRepository) CountGroupByBeanName(userID uint) ([]BeanNameCount, error) {
	rows := make([]BeanNameCount, 0)
	if err := r.db.Model(&model.TastingNote{}).
		Select("coffee_name, COUNT(*) AS count").
		Where("user_id = ?", userID).
		Group("coffee_name").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountGroupByBeanNames returns global note counts grouped by coffee_name,
// restricted to the given names.
func (r *TastingNoteRepository) CountGroupByBeanNames(names []string) (map[string]int64, error) {
	result := make(map[string]int64)
	if len(names) == 0 {
		return result, nil
	}
	rows := make([]BeanNameCount, 0)
	if err := r.db.Model(&model.TastingNote{}).
		Select("coffee_name, COUNT(*) AS count").
		Where("coffee_name IN ?", names).
		Group("coffee_name").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CoffeeName] = row.Count
	}
	return result, nil
}

// CountGroupByBeanNameForUser returns a user's note counts grouped by
// coffee_name, restricted to the given names.
func (r *TastingNoteRepository) CountGroupByBeanNameForUser(userID uint, names []string) ([]BeanNameCount, error) {
	rows := make([]BeanNameCount, 0)
	if len(names) == 0 {
		return rows, nil
	}
	if err := r.db.Model(&model.TastingNote{}).
		Select("coffee_name, COUNT(*) AS count").
		Where("user_id = ? AND coffee_name IN ?", userID, names).
		Group("coffee_name").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountByBeanName returns the number of notes for one coffee name, scoped to a
// user. With userID == 0 it counts notes of all users (public global tally).
func (r *TastingNoteRepository) CountByBeanName(userID uint, coffeeName string) (int64, error) {
	var count int64
	q := r.db.Model(&model.TastingNote{}).Where("coffee_name = ?", coffeeName)
	if userID != 0 {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ListByUserAndBeanName returns a user's notes for one coffee name.
func (r *TastingNoteRepository) ListByUserAndBeanName(userID uint, coffeeName string) ([]model.TastingNote, error) {
	var items []model.TastingNote
	if err := r.db.Where("user_id = ? AND coffee_name = ?", userID, coffeeName).
		Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// AvgScore returns the average overall score of a user's notes.
func (r *TastingNoteRepository) AvgScore(userID uint) (float64, error) {
	var avg float64
	if err := r.db.Model(&model.TastingNote{}).
		Where("user_id = ?", userID).
		Select("COALESCE(AVG(overall_score), 0)").Scan(&avg).Error; err != nil {
		return 0, err
	}
	return avg, nil
}

// TopOrigins returns the top 3 origins by note count for a user.
func (r *TastingNoteRepository) TopOrigins(userID uint) ([]string, error) {
	var origins []string
	if err := r.db.Model(&model.TastingNote{}).
		Where("user_id = ? AND origin <> ''", userID).
		Group("origin").Order("count(*) DESC").Limit(3).
		Pluck("origin", &origins).Error; err != nil {
		return nil, err
	}
	return origins, nil
}
