package models

type ShortenLinkRequest struct {
	LongDurableLink string `json:"longDurableLink"`
}

type ExchangeShortLinkRequest struct {
	RequestedLink string `json:"requestedLink"`
}

type CreateDurableLinkRequest struct {
	DurableLinkInfo DurableLinkInfo `json:"durableLinkInfo"`
	Suffix          Suffix          `json:"suffix,omitempty"`
}
