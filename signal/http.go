package signal

import (
	"log"
	"net/http"
	"encoding/json"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mohit810/streamingcdn/encryptor"
	"github.com/mohit810/streamingcdn/structs"
	"github.com/mohit810/streamingcdn/webrtc"
	"github.com/pion/dtls/v2/examples/util"
)

type Destination struct {
	audioPort int `json:"audioPort"`
	videoPort int `json:"videoPort"`
	destinationType string `json:"destinationType"`
	destinationUrl string `json:"destinationUrl"`
	destinationFormat string `json:"destinationFormat"`
}

// HTTPSDPServer starts a HTTP Server that consumes SDPs
func HTTPSDPServer(r *httprouter.Router) {

	r.ServeFiles("/watch/*filepath", http.Dir("vid"))
	r.POST("/sdp", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		log.Println("Beginning New Session")
		var offer structs.Offer
		err := json.NewDecoder(r.Body).Decode(&offer)
		log.Println("StreamKey: " + offer.StreamKey)
		util.Check(err)

		url := "http://127.0.0.1:8000/config/" + offer.StreamKey + ".json"
		httpClient := http.Client{
			Timeout: time.Second * 2, // Timeout after 2 seconds
		}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Fatal(err)
		}

		res, getErr := httpClient.Do(req)
		if getErr != nil {
			log.Fatal(getErr)
		}

		defer res.Body.Close()

		var destination Destination
		if err := json.NewDecoder(res.Body).Decode(&destination); err != nil {
			log.Println(err)
		}

		answer, err := webrtc.CreateWebRTCConnection(offer.Sdp, offer.StreamKey, destination.audioPort,
			destination.videoPort, destination.destinationType,
			destination.destinationUrl, destination.destinationFormat)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		c := new(structs.Response)
		c.Sdp = encryptor.Encode(answer)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // 201
		err = json.NewEncoder(w).Encode(c)
		util.Check(err)
	})
}
