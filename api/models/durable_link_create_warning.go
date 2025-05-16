package models

// Represents a warning when creating a durable link. For ex. you pass ItunesConnectAnalytics paramters but don't pass an iOS app store id.
type DurableLinkCreationWarning struct {
	WarningCode    string `json:"warningCode"`
	WarningMessage string `json:"warningMessage"`
}
