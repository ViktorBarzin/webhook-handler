package models

type FbMessagePostBackCallback struct {
	Sender struct {
		ID string `json:"id"`
	} `json:"sender"`
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Timestamp int64 `json:"timestamp"`
	Postback  struct {
		Title    string `json:"title"`
		Payload  string `json:"payload"`
		Referral struct {
			Ref    string `json:"ref"`
			Source string `json:"source"`
			Type   string `json:"type"`
		} `json:"referral"`
	} `json:"postback"`
}
