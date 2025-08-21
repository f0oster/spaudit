package ui

import "time"

// ToastNotificationView represents the view model for a toast notification with rich details.
type ToastNotificationView struct {
	Title     string
	Message   string
	Type      string
	JobType   string
	Duration  string
	SiteURL   string
	Stats     *ToastStatsView
	Timestamp time.Time
}

// ToastStatsView represents job statistics for the toast.
type ToastStatsView struct {
	ListsProcessed   int
	ItemsProcessed   int
	PermissionsFound int
	SharingLinks     int
	ErrorsCount      int
}
