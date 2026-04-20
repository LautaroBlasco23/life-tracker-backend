package dto

type FollowResponse struct {
	Status string `json:"status"`
}

type SubscribeRequest struct {
	SourceType string `json:"sourceType" binding:"required,oneof=activity routine"`
	SourceID   uint   `json:"sourceId"   binding:"required,min=1"`
}

type SubscriptionResponse struct {
	SourceType string `json:"sourceType"`
	SourceID   uint   `json:"sourceId"`
	ClonedID   uint   `json:"clonedId"`
	ID         uint   `json:"id"`
}

type FollowerItem struct {
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	UserID    uint   `json:"userId"`
}
