package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
)

type AcademyService struct {
	lessonRepo *repository.LessonRepository
}

func NewAcademyService(lessonRepo *repository.LessonRepository) *AcademyService {
	return &AcademyService{lessonRepo: lessonRepo}
}

func (s *AcademyService) ListLessons(ctx context.Context, filter model.LessonFilter) ([]model.LessonListItem, int, error) {
	return s.lessonRepo.List(ctx, filter)
}

func (s *AcademyService) GetLesson(ctx context.Context, id uuid.UUID) (*model.Lesson, error) {
	return s.lessonRepo.GetByID(ctx, id)
}
