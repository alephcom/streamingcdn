package signal

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/pion/dtls/v2/examples/util"
	"net/http"
	"practice/streamingcdn/encryptor"
	"practice/streamingcdn/structs"
	"practice/streamingcdn/webrtc"
)

// HTTPSDPServer starts a HTTP Server that consumes SDPs
func HTTPSDPServer(r *httprouter.Router) {

	r.ServeFiles("/*filepath", http.Dir("public"))
	r.POST("/sdp", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		var offer structs.Offer
		err := json.NewDecoder(r.Body).Decode(&offer)
		util.Check(err)
		answer, err := webrtc.CreateWebRTCConnection(offer.StreamKey, offer.Sdp)
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
