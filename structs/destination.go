package structs

type Destination struct {
	AudioPort int `json:"audioPort"`
	VideoPort int `json:"videoPort"`
	DestinationType string `json:"destinationType"`
	DestinationUrl string `json:"destinationUrl"`
	DestinationFormat string `json:"destinationFormat"`
}
