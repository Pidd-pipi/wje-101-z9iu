package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// NoteService handles tasting notes.
type NoteService struct {
	repo     *repository.TastingNoteRepository
	beanRepo *repository.CoffeeBeanRepository
	listRepo *repository.UserBeanListRepository
	logger   *slog.Logger
}

// NewNoteService creates a NoteService.
func NewNoteService(repo *repository.TastingNoteRepository, beanRepo *repository.CoffeeBeanRepository, listRepo *repository.UserBeanListRepository, logger *slog.Logger) *NoteService {
	return &NoteService{repo: repo, beanRepo: beanRepo, listRepo: listRepo, logger: logger}
}

// ensureBeanListed tracks the bean matching the note's coffee name for the
// user, so a later delete / bean switch that brings the note count back to zero
// moves the bean into the want-to-drink group. Best effort by design.
func (s *NoteService) ensureBeanListed(userID uint, coffeeName string) {
	if coffeeName == "" {
		return
	}
	bean, err := s.beanRepo.FindByName(coffeeName)
	if err != nil {
		return
	}
	if err := s.listRepo.Add(userID, bean.ID); err != nil {
		s.logger.Warn("note create ensure bean list entry failed", "user_id", userID, "bean_id", bean.ID, "error", err)
	}
}

// Create adds a note for a user.
func (s *NoteService) Create(userID uint, n *model.TastingNote) (*model.TastingNote, error) {
	if !constants.IsValidRoastLevel(n.RoastLevel) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("TastingNote[roast_level=%s] create failed: invalid roast level", n.RoastLevel))
	}
	n.UserID = userID
	if n.FlavorTags == "" {
		n.FlavorTags = "[]"
	}
	if err := s.repo.Create(n); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogNoteCreateFailed, n.CoffeeName), "error", err)
		return nil, fmt.Errorf("note create: %w", err)
	}
	s.ensureBeanListed(userID, n.CoffeeName)
	s.logger.Info(fmt.Sprintf(constants.LogNoteCreateSuccess, n.CoffeeName), "id", n.ID)
	return n, nil
}

// Get returns a note by id.
func (s *NoteService) Get(id uint) (*model.TastingNote, error) {
	n, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("TastingNote[id=%d] not found", id))
		}
		return nil, fmt.Errorf("note get: %w", err)
	}
	return n, nil
}

// Update edits a note owned by the user. Switching the selected bean decrements
// the old bean's note count and increments the new one; the counts are derived,
// so only list-entry bookkeeping is needed.
func (s *NoteService) Update(userID, id uint, n *model.TastingNote) (*model.TastingNote, error) {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("TastingNote[id=%d] update failed: not found", id))
		}
		return nil, fmt.Errorf("note update find: %w", err)
	}
	if exist.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("TastingNote[id=%d] update failed: user_id=%d not owner", id, userID))
	}
	if n.RoastLevel != "" {
		if !constants.IsValidRoastLevel(n.RoastLevel) {
			return nil, util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("TastingNote[id=%d] update failed: invalid roast level %s", id, n.RoastLevel))
		}
		exist.RoastLevel = n.RoastLevel
	}
	if n.CoffeeName != "" {
		exist.CoffeeName = n.CoffeeName
	}
	exist.Origin = n.Origin
	exist.FlavorTags = n.FlavorTags
	if exist.FlavorTags == "" {
		exist.FlavorTags = "[]"
	}
	exist.AromaScore = n.AromaScore
	exist.AcidityScore = n.AcidityScore
	exist.BodyScore = n.BodyScore
	exist.OverallScore = n.OverallScore
	exist.BrewMethod = n.BrewMethod
	exist.BrewRecipeID = n.BrewRecipeID
	exist.NotesText = n.NotesText
	exist.ImageURL = n.ImageURL
	if err := s.repo.Update(exist); err != nil {
		return nil, fmt.Errorf("note update: %w", err)
	}
	s.ensureBeanListed(userID, exist.CoffeeName)
	s.logger.Info(fmt.Sprintf(constants.LogNoteUpdateSuccess, id), "id", id)
	return exist, nil
}

// Delete removes a note owned by the user. The bean stays tracked, returning
// it to want-to-drink when this was the user's last matching note.
func (s *NoteService) Delete(userID, id uint) error {
	n, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("TastingNote[id=%d] delete failed: not found", id))
		}
		return fmt.Errorf("note delete find: %w", err)
	}
	if n.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("TastingNote[id=%d] delete failed: not owner", id))
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("note delete: %w", err)
	}
	s.ensureBeanListed(userID, n.CoffeeName)
	s.logger.Info(fmt.Sprintf(constants.LogNoteDeleteSuccess, id), "id", id)
	return nil
}

// List filters notes.
func (s *NoteService) List(roast, origin, keyword string, hot bool, page, pageSize int) ([]model.TastingNote, int64, error) {
	items, total, err := s.repo.List(roast, origin, keyword, hot, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("note list: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogNoteListSuccess, roast, page), "total", total)
	return items, total, nil
}

// ListByUser returns notes of a user.
func (s *NoteService) ListByUser(userID uint) ([]model.TastingNote, error) {
	items, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("note list by user: %w", err)
	}
	return items, nil
}

// AvgScore returns the average overall score of a user's notes.
func (s *NoteService) AvgScore(userID uint) (float64, error) {
	avg, err := s.repo.AvgScore(userID)
	if err != nil {
		return 0, fmt.Errorf("note avg score: %w", err)
	}
	return avg, nil
}

// TopOrigins returns the top 3 origins by note count.
func (s *NoteService) TopOrigins(userID uint) ([]string, error) {
	origins, err := s.repo.TopOrigins(userID)
	if err != nil {
		return nil, fmt.Errorf("note top origins: %w", err)
	}
	return origins, nil
}
