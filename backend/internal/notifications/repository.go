package notifications

import "context"

type Model struct {
	ID        int64
	UserID    int64
	Title     string
	Message   string
	Type      string
	IsRead    bool
	CreatedAt string
}

type Repository interface {
	Create(ctx context.Context, userID int64, title, message, notificationType string) error
	List(ctx context.Context, userID int64) ([]Model, error)
	MarkAsRead(ctx context.Context, notificationID, userID int64) error
}
