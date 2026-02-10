package dto

type ResponseMessage struct {
	Message string `json:"message"`
}

type ResponseHealth struct {
	Status string `json:"status"`
}
