package model

import "image"

type BulkRecognizeRequest struct {
	Images              [][]byte `json:"payload"`
	IncludeFacesDetails bool     `json:"includeFacesDetails"`
	RequestId           string   `json:"requestId"`
	ResponseTo          string   `json:"responseTo"`
}

type BulkRecognizeResponse struct {
	Labels          []string        `json:"labels"`
	FaceDetailsList [][]FaceDetails `json:"faceDetailsList"`
	Error           error           `json:"error"`
}

type FaceDetails struct {
	Rect       image.Rectangle `json:"rect"`
	Descriptor [128]float32    `json:"descriptor"`
	Image      []byte          `json:"image"`
}

