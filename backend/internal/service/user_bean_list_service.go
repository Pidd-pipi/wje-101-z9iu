package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/dto"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// UserBeanListService manages a user's "want to drink" list and derives the
// "want to drink" / "already drunk" groups for the personal profile.
type UserBeanListService struct {
	listRepo *repository.UserBeanListRepository
	beanRepo *repository.CoffeeBeanRepository
	noteRepo *repository.TastingNoteRepository
	logger   *slog.Logger
}

// NewUserBeanListService creates a UserBeanListService.
func NewUserBeanListService(listRepo *repository.UserBeanListRepository, beanRepo *repository.CoffeeBeanRepository, noteRepo *repository.TastingNoteRepository, logger *slog.Logger) *UserBeanListService {
	return &UserBeanListService{listRepo: listRepo, beanRepo: beanRepo, noteRepo: noteRepo, logger: logger}
}

// Add puts a bean into the user's want-to-drink list (one entry per bean).
// Adding a bean that is already tracked or already drunk is idempotent.
func (s *UserBeanListService) Add(userID, beanID uint) error {
	if _, err := s.beanRepo.FindByID(beanID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("CoffeeBean[id=%d] add to list failed: bean not found", beanID))
		}
		return fmt.Errorf("bean list add find bean: %w", err)
	}
	if err := s.listRepo.Add(userID, beanID); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogBeanListAddFailed, userID, beanID), "error", err)
		return fmt.Errorf("bean list add: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogBeanListAddSuccess, userID, beanID))
	return nil
}

// Remove moves a bean out of the user's want-to-drink list. A bean the user
// has already drunk cannot be removed here.
func (s *UserBeanListService) Remove(userID, beanID uint) error {
	bean, err := s.beanRepo.FindByID(beanID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("CoffeeBean[id=%d] remove from list failed: bean not found", beanID))
		}
		return fmt.Errorf("bean list remove find bean: %w", err)
	}
	count, err := s.noteRepo.CountByBeanName(userID, bean.Name)
	if err != nil {
		return fmt.Errorf("bean list remove count: %w", err)
	}
	if count > 0 {
		return util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("UserBeanList[user_id=%d bean_id=%d] remove failed: already drunk with %d note(s)", userID, beanID, count))
	}
	if err := s.listRepo.Delete(userID, beanID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("UserBeanList[user_id=%d bean_id=%d] not found", userID, beanID))
		}
		return fmt.Errorf("bean list remove: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogBeanListRemoveSuccess, userID, beanID))
	return nil
}

// GetUserBeanGroups splits the user's beans into want-to-drink and drunk.
// A bean is drunk when the user has at least one note with its name; when the
// count returns to zero the tracked entry falls back to want-to-drink.
func (s *UserBeanListService) GetUserBeanGroups(userID uint) (*dto.UserBeanGroupData, error) {
	trackedIDs, err := s.listRepo.ListBeanIDsByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("bean groups list tracked: %w", err)
	}
	countRows, err := s.noteRepo.CountGroupByBeanName(userID)
	if err != nil {
		return nil, fmt.Errorf("bean groups count notes: %w", err)
	}
	myCountByName := make(map[string]int64, len(countRows))
	names := make([]string, 0, len(trackedIDs)+len(countRows))
	for _, row := range countRows {
		myCountByName[row.CoffeeName] = row.Count
		names = append(names, row.CoffeeName)
	}

	// Tracked beans form the base; beans carrying notes are unioned in so a
	// published note always shows as drunk even without a list row.
	beanByID := make(map[uint]*model.CoffeeBean)
	for _, id := range trackedIDs {
		b, err := s.beanRepo.FindByID(id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("bean groups find tracked: %w", err)
		}
		beanByID[b.ID] = b
		names = append(names, b.Name)
	}
	for name := range myCountByName {
		b, err := s.beanRepo.FindByName(name)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("bean groups find by name: %w", err)
		}
		beanByID[b.ID] = b
	}

	globalCounts, err := s.noteRepo.CountGroupByBeanNames(names)
	if err != nil {
		return nil, fmt.Errorf("bean groups global counts: %w", err)
	}

	group := &dto.UserBeanGroupData{Want: []dto.BeanListItem{}, Drunk: []dto.BeanListItem{}}
	// Keep the tracking order for want items and note name order for drunk.
	for _, id := range trackedIDs {
		b, ok := beanByID[id]
		if !ok {
			continue
		}
		if myCountByName[b.Name] > 0 {
			continue
		}
		group.Want = append(group.Want, dto.BeanListItem{
			CoffeeBean: *b, InList: true,
			NoteCount: globalCounts[b.Name], MyNoteCount: 0,
		})
	}
	for _, row := range countRows {
		if row.Count == 0 {
			continue
		}
		b, err := s.beanRepo.FindByName(row.CoffeeName)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("bean groups drunk find: %w", err)
		}
		group.Drunk = append(group.Drunk, dto.BeanListItem{
			CoffeeBean: *b, InList: true,
			NoteCount: globalCounts[b.Name], MyNoteCount: row.Count,
		})
	}
	s.logger.Info(fmt.Sprintf(constants.LogBeanListGroupSuccess, userID), "want", len(group.Want), "drunk", len(group.Drunk))
	return group, nil
}
