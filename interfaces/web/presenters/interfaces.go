// Package presenters interfaces defines contracts for presenter components.
package presenters

import (
	"spaudit/application"
)

// ListPresenterInterface defines the contract for list presenters.
type ListPresenterInterface interface {
	// ToSiteListsViewModel converts service data to view model.
	ToSiteListsViewModel(data *application.SiteWithListsData) *SiteListsVM
}

// Ensure ListPresenter implements the interface.
var _ ListPresenterInterface = (*ListPresenter)(nil)
