package structs

type Destination struct {
	AudioPort int `json:"audioPort"`
	AudioEnable bool `json:"audioEnable"`
	VideoPort int `json:"videoPort"`
	VideoEnable bool `json:"videoEnable"`
	DestinationType string `json:"destinationType"`
	DestinationUrl string `json:"destinationUrl"`
	DestinationFormat string `json:"destinationFormat"`
}
