package airtablewatcher

// AirtableAttachment is a single airtable attachment
type AirtableAttachment struct {
	// unique attachment id
	ID string `json:"id,omitempty"`
	// url, e.g. "https://dl.airtable.com/foo.jpg"
	URL string `json:"url,omitempty"`
	// filename, e.g. "foo.jpg"
	Filename string `json:"filename,omitempty"`
	// file size, in bytes
	Size int64 `json:"size,omitempty"`
	// content type, e.g. "image/jpeg"
	Type string `json:"type,omitempty"`
	// Width in pixels
	Width int64 `json:"width,omitempty"`
	// Height in pixels
	Height int64 `json:"height,omitempty"`
}

// AirtableAttachments is a structure used to upload attachments to airtable.
//
// It can be used in an Update/Set request, or creating a record
type AirtableAttachments []AirtableAttachment
