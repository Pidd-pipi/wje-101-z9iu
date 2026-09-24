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

// UserBeanService manages each user's to-drink / tasted bean list.
type UserBeanService struct {
	repo     *repository.UserBeanRepository
	beanRepo *repository.CoffeeBeanRepository
	noteRepo *repository.TastingNoteRepository
	logger   *slog.Logger
}

// NewUserBeanService creates a UserBeanService.
func NewUserBeanService(repo *repository.UserBeanRepository, beanRepo *repository.CoffeeBeanRepository, noteRepo *repository.TastingNoteRepository, logger *slog.Logger) *UserBeanService {
	return &UserBeanService{repo: repo, beanRepo: beanRepo, noteRepo: noteRepo, logger: logger}
}

// Add puts a bean into the user's to-drink list (one row per bean).
func (s *UserBeanService) Add(userID, beanID uint) (*model.UserBean, error) {
	if _, err := s.beanRepo.FindByID(beanID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CoffeeBean[id=%d] not found", beanID))
		}
		return nil, fmt.Errorf("user bean add find bean: %w", err)
	}
	ub := &model.UserBean{UserID: userID, BeanID: beanID}
	if err := s.repo.Create(ub); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("UserBean[user_id=%d bean_id=%d] add failed: %s", userID, beanID, constants.MsgBeanWantConflict))
		}
		s.logger.Error(fmt.Sprintf(constants.LogUserBeanAddFailed, userID, beanID), "error", err)
		return nil, fmt.Errorf("user bean add: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogUserBeanAddSuccess, userID, beanID), "id", ub.ID)
	return ub, nil
}

// Remove moves a bean out of the list; only a 待喝 bean (zero notes) may be removed.
func (s *UserBeanService) Remove(userID, beanID uint) error {
	if _, err := s.repo.Find(userID, beanID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("UserBean[user_id=%d bean_id=%d] not found", userID, beanID))
		}
		return fmt.Errorf("user bean remove find: %w", err)
	}
	count, err := s.noteRepo.CountByUserBean(userID, beanID)
	if err != nil {
		return fmt.Errorf("user bean remove count: %w", err)
	}
	if count > 0 {
		s.logger.Warn(fmt.Sprintf(constants.LogUserBeanRemoveFailed, userID, beanID), "note_count", count)
		return util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("UserBean[user_id=%d bean_id=%d] remove failed: %s (note_count=%d)", userID, beanID, constants.MsgBeanTastedBlocked, count))
	}
	if err := s.repo.Delete(userID, beanID); err != nil {
		return fmt.Errorf("user bean remove: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogUserBeanRemoveSuccess, userID, beanID))
	return nil
}

// EnsureTracked makes sure a tracking row exists; it is invoked whenever a
// user publishes or edits a tasting note onto a bean, turning it 喝过.
// It validates the bean exists so notes never dangle an orphan tracking row.
func (s *UserBeanService) EnsureTracked(userID, beanID uint) error {
	if beanID == 0 {
		return nil
	}
	if _, err := s.beanRepo.FindByID(beanID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("TastingNote[bean_id=%d] save failed: bean not found", beanID))
		}
		return fmt.Errorf("user bean ensure bean: %w", err)
	}
	if _, err := s.repo.Find(userID, beanID); err == nil {
		return nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("user bean ensure find: %w", err)
	}
	if err := s.repo.Create(&model.UserBean{UserID: userID, BeanID: beanID}); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil
		}
		return fmt.Errorf("user bean ensure create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogUserBeanTracked, userID, beanID))
	return nil
}

// DecorateList annotates a bean list with public note totals and, when
// viewerID is non-zero, that viewer's tracking state.
func (s *UserBeanService) DecorateList(beans []model.CoffeeBean, viewerID uint) ([]dto.BeanListItem, error) {
	publicCounts, err := s.noteRepo.CountByBeans()
	if err != nil {
		return nil, fmt.Errorf("user bean decorate counts: %w", err)
	}
	trackedIDs := map[uint]bool{}
	userCounts := map[uint]int64{}
	if viewerID != 0 {
		ids, err := s.repo.BeanIDsByUser(viewerID)
		if err != nil {
			return nil, fmt.Errorf("user bean decorate ids: %w", err)
		}
		for _, id := range ids {
			trackedIDs[id] = true
		}
		userCounts, err = s.noteRepo.CountByUserBeans(viewerID)
		if err != nil {
			return nil, fmt.Errorf("user bean decorate user counts: %w", err)
		}
	}
	items := make([]dto.BeanListItem, 0, len(beans))
	for _, b := range beans {
		item := dto.BeanListItem{CoffeeBean: b, NoteTotal: publicCounts[b.ID]}
		if viewerID != 0 && trackedIDs[b.ID] {
			item.Tracked = true
			item.UserNoteCount = userCounts[b.ID]
			item.TrackStatus = constants.BeanStatusWant
			if userCounts[b.ID] > 0 {
				item.TrackStatus = constants.BeanStatusTasted
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// ProfileGroups splits a user's tracked beans into 待喝 and 喝过 groups.
func (s *UserBeanService) ProfileGroups(userID uint) (*dto.UserBeanGroups, error) {
	ids, err := s.repo.BeanIDsByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("user bean groups ids: %w", err)
	}
	beans, err := s.beanRepo.FindByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("user bean groups beans: %w", err)
	}
	byID := make(map[uint]model.CoffeeBean, len(beans))
	for _, b := range beans {
		byID[b.ID] = b
	}
	counts, err := s.noteRepo.CountByUserBeans(userID)
	if err != nil {
		return nil, fmt.Errorf("user bean groups counts: %w", err)
	}
	groups := &dto.UserBeanGroups{Want: []dto.UserBeanGroupItem{}, Tasted: []dto.UserBeanGroupItem{}}
	// Preserve oldest-first ordering from BeanIDsByUser.
	for _, id := range ids {
		b, ok := byID[id]
		if !ok {
			continue
		}
		item := dto.UserBeanGroupItem{CoffeeBean: b, NoteCount: counts[id], Notes: []model.TastingNote{}}
		if counts[id] > 0 {
			item.Notes, err = s.noteRepo.ListByUserBean(userID, id)
			if err != nil {
				return nil, fmt.Errorf("user bean groups notes: %w", err)
			}
			groups.Tasted = append(groups.Tasted, item)
		} else {
			groups.Want = append(groups.Want, item)
		}
	}
	s.logger.Info(fmt.Sprintf(constants.LogUserBeanGroupSuccess, userID, len(groups.Want), len(groups.Tasted)))
	return groups, nil
}
