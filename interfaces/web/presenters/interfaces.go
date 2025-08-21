package presenters

import (
	"spaudit/application"
)

// ListPresenterInterface defines the contract for list presentation logic.
type ListPresenterInterface interface {
	// ToSiteListsViewModel converts service data to SiteListsVM view model.
	ToSiteListsViewModel(data *application.SiteWithListsData) *SiteListsVM
}

// Ensure ListPresenter implements the interface.
var _ ListPresenterInterface = (*ListPresenter)(nil)
