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

// BeanService handles coffee bean library.
type BeanService struct {
	repo     *repository.CoffeeBeanRepository
	noteRepo *repository.TastingNoteRepository
	listRepo *repository.UserBeanListRepository
	logger   *slog.Logger
}

// NewBeanService creates a BeanService.
func NewBeanService(repo *repository.CoffeeBeanRepository, noteRepo *repository.TastingNoteRepository, listRepo *repository.UserBeanListRepository, logger *slog.Logger) *BeanService {
	return &BeanService{repo: repo, noteRepo: noteRepo, listRepo: listRepo, logger: logger}
}

// Create adds a bean (admin).
func (s *BeanService) Create(b *model.CoffeeBean) (*model.CoffeeBean, error) {
	if !constants.IsValidProcessMethod(b.ProcessMethod) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CoffeeBean[process_method=%s] create failed: invalid process method", b.ProcessMethod))
	}
	if b.FlavorTags == "" {
		b.FlavorTags = "[]"
	}
	if err := s.repo.Create(b); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CoffeeBean[name=%s] create failed: name exists", b.Name))
		}
		s.logger.Error(fmt.Sprintf(constants.LogBeanCreateFailed, b.Name), "error", err)
		return nil, fmt.Errorf("bean create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogBeanCreateSuccess, b.Name), "id", b.ID)
	return b, nil
}

// Update edits a bean (admin).
func (s *BeanService) Update(id uint, b *model.CoffeeBean) (*model.CoffeeBean, error) {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("bean update find: %w", err)
	}
	if b.Name != "" {
		exist.Name = b.Name
	}
	if b.Origin != "" {
		exist.Origin = b.Origin
	}
	if b.ProcessMethod != "" {
		if !constants.IsValidProcessMethod(b.ProcessMethod) {
			return nil, util.NewAppError(422, constants.CodeValidationError, "invalid process method")
		}
		exist.ProcessMethod = b.ProcessMethod
	}
	if b.FlavorTags != "" {
		exist.FlavorTags = b.FlavorTags
	}
	if b.Description != "" {
		exist.Description = b.Description
	}
	if err := s.repo.Update(exist); err != nil {
		return nil, fmt.Errorf("bean update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogBeanUpdateSuccess, id), "id", id)
	return exist, nil
}

// Delete removes a bean (admin) and clears user list entries pointing at it.
// List rows are removed first because the SQL schema declares a foreign key
// from user_bean_lists to coffee_beans.
func (s *BeanService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		return fmt.Errorf("bean delete find: %w", err)
	}
	if err := s.listRepo.DeleteByBean(id); err != nil {
		return fmt.Errorf("bean delete list cleanup: %w", err)
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("bean delete: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogBeanDeleteSuccess, id), "id", id)
	return nil
}

// List filters beans. viewerID == 0 means anonymous browsing: only the public
// tasting-note tally is attached; otherwise personal list state is attached.
func (s *BeanService) List(origin, process, keyword string, page, pageSize int, viewerID uint) ([]dto.BeanListItem, int64, error) {
	items, total, err := s.repo.List(origin, process, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("bean list: %w", err)
	}
	result := make([]dto.BeanListItem, 0, len(items))
	if len(items) == 0 {
		s.logger.Info(fmt.Sprintf(constants.LogBeanListSuccess, process), "total", total)
		return result, total, nil
	}

	names := make([]string, 0, len(items))
	for _, b := range items {
		names = append(names, b.Name)
	}
	globalCounts, err := s.noteRepo.CountGroupByBeanNames(names)
	if err != nil {
		return nil, 0, fmt.Errorf("bean list counts: %w", err)
	}
	tracked := make(map[uint]bool)
	myCounts := make(map[string]int64)
	if viewerID != 0 {
		trackedIDs, err := s.listRepo.ListBeanIDsByUser(viewerID)
		if err != nil {
			return nil, 0, fmt.Errorf("bean list user list: %w", err)
		}
		for _, id := range trackedIDs {
			tracked[id] = true
		}
		myRows, err := s.noteRepo.CountGroupByBeanNameForUser(viewerID, names)
		if err != nil {
			return nil, 0, fmt.Errorf("bean list user counts: %w", err)
		}
		for _, row := range myRows {
			myCounts[row.CoffeeName] = row.Count
		}
	}
	for _, b := range items {
		result = append(result, dto.BeanListItem{
			CoffeeBean:  b,
			NoteCount:   globalCounts[b.Name],
			MyNoteCount: myCounts[b.Name],
			InList:      tracked[b.ID],
		})
	}
	s.logger.Info(fmt.Sprintf(constants.LogBeanListSuccess, process), "total", total)
	return result, total, nil
}
