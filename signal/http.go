package signal

import (
	"io/ioutil"
	"log"
	"net/http"
	"encoding/json"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/alephcom/streamingcdn/encryptor"
	"github.com/alephcom/streamingcdn/structs"
	"github.com/alephcom/streamingcdn/webrtc"
	"github.com/pion/dtls/v2/examples/util"
)



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
		log.Println("Url: " + url)
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

		if res.Body != nil {
			defer res.Body.Close()
		}

		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			log.Fatal(readErr)
		}
		log.Println(body)

		destination := structs.Destination{}
		jsonErr := json.Unmarshal(body, &destination)
		if jsonErr != nil {
			log.Fatal(jsonErr)
		}

		answer, err := webrtc.CreateWebRTCConnection(offer.Sdp, offer.StreamKey, destination)

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
