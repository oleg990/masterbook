package admin

import (
	"context"
	"errors"
	"testing"

	"masterbook/internal/auth"
	"masterbook/internal/masters"
)

type fakeMasterCreator struct {
	err error
}

func (f fakeMasterCreator) RegisterWithRole(ctx context.Context, name, email, password, role string) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	return auth.User{ID: 10, Name: name, Email: email, Role: role}, nil
}

type fakeProfileCreator struct{}

func (fakeProfileCreator) CreateProfile(ctx context.Context, userID int64, description string, photoURL *string) (masters.Profile, error) {
	return masters.Profile{ID: 20, UserID: userID, Description: description, PhotoURL: photoURL}, nil
}

func TestCreateMasterReturnsCombinedResult(t *testing.T) {
	s := NewService(fakeMasterCreator{}, fakeProfileCreator{})
	result, err := s.CreateMaster(context.Background(), "Иван", "ivan@example.com", "password123", "Стрижки", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.UserID != 10 || result.ProfileID != 20 || result.Name != "Иван" || result.Email != "ivan@example.com" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCreateMasterPropagatesEmailExists(t *testing.T) {
	s := NewService(fakeMasterCreator{err: auth.ErrEmailExists}, fakeProfileCreator{})
	_, err := s.CreateMaster(context.Background(), "Иван", "ivan@example.com", "password123", "Стрижки", nil)
	if !errors.Is(err, auth.ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
}
