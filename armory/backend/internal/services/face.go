package services

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"

	"armory/internal/database"
	"armory/internal/models"
)

const (
	descriptorLen  = 128
	maxEnrollFaces = 20
	faceModelName  = "face-api-1.7.15"
	MatchThreshold = 0.5
	noDistance     = math.MaxFloat64
)

type FaceService struct {
	store *database.Store
}

func NewFaceService(store *database.Store) *FaceService {
	return &FaceService{store: store}
}

type MatchResult struct {
	Matched  bool
	User     models.User
	Distance float64
}

func (s *FaceService) PendingEnrollments(ctx context.Context) ([]models.FaceEnrollmentRequest, error) {
	return s.store.ListPendingFaceEnrollmentRequests(ctx)
}

func (s *FaceService) DecideEnrollment(ctx context.Context, id, adminID int64, approve bool) error {
	err := s.store.DecideFaceEnrollmentRequest(ctx, id, adminID, approve)
	switch {
	case errors.Is(err, database.ErrNotFound):
		return ErrEnrollmentNotFound
	case errors.Is(err, database.ErrConflict):
		return ErrEnrollmentHandled
	case errors.Is(err, database.ErrDuplicate):
		return ErrUserExists
	default:
		return err
	}
}

func validDescriptor(d []float64) bool {
	if len(d) != descriptorLen {
		return false
	}
	for _, v := range d {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

func distance(a, b []float64) float64 {
	var sum float64
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

func (s *FaceService) Enroll(ctx context.Context, userID int64, descriptors [][]float64) error {
	if len(descriptors) == 0 || len(descriptors) > maxEnrollFaces {
		return ErrInvalidDescriptor
	}
	refs := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		if !validDescriptor(d) {
			return ErrInvalidDescriptor
		}
		b, err := json.Marshal(d)
		if err != nil {
			return err
		}
		refs = append(refs, string(b))
	}

	err := s.store.ReplaceFaceEnrollments(ctx, userID, refs, faceModelName)
	if errors.Is(err, database.ErrNotFound) {
		return ErrUserNotFound
	}
	return err
}

func (s *FaceService) RequestEnrollment(ctx context.Context, name, serviceNo string, descriptors [][]float64) error {
	name, serviceNo = strings.TrimSpace(name), strings.ToUpper(strings.TrimSpace(serviceNo))
	if name == "" {
		return ErrNameRequired
	}
	if serviceNo == "" {
		return ErrServiceNoRequired
	}
	refs, err := encodeDescriptors(descriptors)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(refs)
	if err != nil {
		return err
	}
	if err := s.store.CreateFaceEnrollmentRequest(ctx, name, serviceNo, string(raw), faceModelName); errors.Is(err, database.ErrDuplicate) {
		return ErrEnrollmentPending
	} else {
		return err
	}
}

func encodeDescriptors(descriptors [][]float64) ([]string, error) {
	if len(descriptors) == 0 || len(descriptors) > maxEnrollFaces {
		return nil, ErrInvalidDescriptor
	}
	refs := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		if !validDescriptor(d) {
			return nil, ErrInvalidDescriptor
		}
		b, err := json.Marshal(d)
		if err != nil {
			return nil, err
		}
		refs = append(refs, string(b))
	}
	return refs, nil
}

func (s *FaceService) Match(ctx context.Context, descriptor []float64) (MatchResult, error) {
	if !validDescriptor(descriptor) {
		return MatchResult{}, ErrInvalidDescriptor
	}
	enrollments, err := s.store.ListFaceEnrollments(ctx)
	if err != nil {
		return MatchResult{}, err
	}

	best := noDistance
	var bestUser models.User
	for _, e := range enrollments {
		var ref []float64
		if json.Unmarshal([]byte(e.FaceRef), &ref) != nil || !validDescriptor(ref) {
			continue
		}
		if d := distance(descriptor, ref); d < best {
			best = d
			bestUser = e.User
		}
	}
	if best > MatchThreshold {
		return MatchResult{Distance: best}, nil
	}
	return MatchResult{Matched: true, User: bestUser, Distance: best}, nil
}
