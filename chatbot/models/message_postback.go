package models

type PayloadPostback struct {
	Recipient Recipient           `json:"recipient"`
	Message   MessageWithPostback `json:"message"`
}

type MessageWithPostback struct {
	Attachment MessageWithPostbackAttachment `json:"attachment"`
}

type MessageWithPostbackAttachment struct {
	Type    string                     `json:"type"`
	Payload MessageWithPostbackPayload `json:"payload"`
}

type MessageWithPostbackPayload struct {
	TemplateType string                       `json:"template_type"`
	Elements     []MessageWithPostbackElement `json:"elements"`
}

type MessageWithPostbackElement struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	// ImageURL string                      `json:"image_url"`
	Buttons []MessageWithPostbackButton `json:"buttons"`
}

type MessageWithPostbackButton struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Payload string `json:"payload"`
}

type FbMessagePostBackCallback struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string `json:"id"`
		Time      int64  `json:"time"`
		Messaging []struct {
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			Recipient struct {
				ID string `json:"id"`
			} `json:"recipient"`
			Timestamp int64 `json:"timestamp"`
			Postback  struct {
				Title   string `json:"title"`
				Payload string `json:"payload"`
			} `json:"postback"`
		} `json:"messaging"`
	} `json:"entry"`
}
