package models

type Payload struct {
	Recipient Recipient `json:"recipient"`
	Message   Message   `json:"message"`
}

type Recipient struct {
	ID string `json:"id"`
}
type Message struct {
	Text string `json:"text"`
}

type FbMessageCallback struct {
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
			Message   struct {
				Mid  string `json:"mid"`
				Text string `json:"text"`
				Nlp  struct {
					Intents  []interface{} `json:"intents"`
					Entities struct {
						WitLocationLocation []struct {
							ID         string        `json:"id"`
							Name       string        `json:"name"`
							Role       string        `json:"role"`
							Start      int           `json:"start"`
							End        int           `json:"end"`
							Body       string        `json:"body"`
							Confidence float64       `json:"confidence"`
							Entities   []interface{} `json:"entities"`
							Suggested  bool          `json:"suggested"`
							Value      string        `json:"value"`
							Type       string        `json:"type"`
						} `json:"wit$location:location"`
					} `json:"entities"`
					Traits struct {
						WitSentiment []struct {
							ID         string  `json:"id"`
							Value      string  `json:"value"`
							Confidence float64 `json:"confidence"`
						} `json:"wit$sentiment"`
						WitGreetings []struct {
							ID         string  `json:"id"`
							Value      string  `json:"value"`
							Confidence float64 `json:"confidence"`
						} `json:"wit$greetings"`
					} `json:"traits"`
					DetectedLocales []struct {
						Locale     string  `json:"locale"`
						Confidence float64 `json:"confidence"`
					} `json:"detected_locales"`
				} `json:"nlp"`
			} `json:"message"`
		} `json:"messaging"`
	} `json:"entry"`
}
