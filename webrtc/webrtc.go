package webrtc

import (
	"context"
	"fmt"
	"github.com/mohit810/streamingcdn/ffmpeg"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/mohit810/streamingcdn/encryptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type udpConn struct {
	conn *net.UDPConn
	port int
}

// WriteToFile will print any string of text to a file safely by
// checking for errors and syncing at the end.
func WriteToFile(filename string, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	return file.Sync()
}

// CreateWebRTCConnection function
func CreateWebRTCConnection(offerStr, streamKey string, audioPort int, videoPort int, outputType string, outputLocation string, outputFormat string) (answer webrtc.SessionDescription, err error) {

	defer func() {
		if e, ok := recover().(error); ok {
			err = e
			err = e
		}
	}()

	// Create a MediaEngine object to configure the supported codec
	m := webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/h264", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        102,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	// Create the API object with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&m))

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Allow us to receive 1 audio track, and 1 video track
	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	} else if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	go func(peerConnection *webrtc.PeerConnection) {
		// Create context
		ctx, cancel := context.WithCancel(context.Background())

		// Create a local addr
		var laddr *net.UDPAddr
		if laddr, err = net.ResolveUDPAddr("udp", "127.0.0.1:"); err != nil {
			fmt.Println(err)
			cancel()
		}

		goDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		errDir := os.MkdirAll("sdp", 0755)
		if errDir != nil {
			log.Fatal(errDir)
		}

		var sdpFile = "sdp/" + streamKey + ".sdp"
		errWrite := WriteToFile(goDir + "/" + sdpFile,
		"v=0\n" +
		 "o=- 0 0 IN IP4 127.0.0.1\n" +
		 "s=Pion WebRTC\n" +
		 "c=IN IP4 127.0.0.1\n" +
		 "t=0 0\n" +
		 "m=audio " + strconv.Itoa( audioPort) +  " RTP/AVP 111\n" +
		 "a=rtpmap:111 OPUS/48000/2\n" +
		 "m=video " + strconv.Itoa( videoPort) + " RTP/AVP 102\n" +
		 "a=rtpmap:102 H264/90000")

		if errWrite != nil {
			log.Fatal(errWrite)
		}

		// Prepare udp conns
		udpConns := map[string]*udpConn{
			"audio": {port: audioPort},
			"video": {port: videoPort},
		}

		for _, c := range udpConns {
			// Create remote addr
			var raddr *net.UDPAddr
			if raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", c.port)); err != nil {
				fmt.Println(err)
				cancel()
			}

			// Dial udp
			if c.conn, err = net.DialUDP("udp", laddr, raddr); err != nil {
				fmt.Println(err)
				cancel()
			}
			defer func(conn net.PacketConn) {
				if closeErr := conn.Close(); closeErr != nil {
					fmt.Println(closeErr)
				}
			}(c.conn)
		}

		ffmpeg.StartFFmpeg(ctx, streamKey, sdpFile, outputType, outputLocation, outputFormat)

		// Set a handler for when a new remote track starts, this handler will forward data to
		// our UDP listeners.
		// In your application this is where you would handle/process audio/video
		peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			fmt.Println("on track called")

			// Retrieve udp connection
			c, ok := udpConns[track.Kind().String()]
			if !ok {
				return
			}

			// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
			go func() {
				ticker := time.NewTicker(time.Second * 2)
				for range ticker.C {
					if rtcpErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}}); rtcpErr != nil {
						fmt.Println(rtcpErr)
					}
					if rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.ReceiverEstimatedMaximumBitrate{Bitrate: 1500000, SenderSSRC: uint32(track.SSRC())}}); rtcpSendErr != nil {
						fmt.Println(rtcpSendErr)
					}
					if ctx.Err() == context.Canceled {
						break
					}
				}
			}()

			b := make([]byte, 1500)
			for {
				// Read
				n, readErr := track.Read(b)
				if readErr != nil {
					fmt.Println(readErr)
				}

				// Write
				if _, err = c.conn.Write(b[:n]); err != nil {
					fmt.Println(err)
					if ctx.Err() == context.Canceled {
						break
					}
				}
			}
		})

		// in a production application you should exchange ICE Candidates via OnICECandidate
		peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			fmt.Println(candidate)
		})

		// Set the handler for ICE connection state
		// This will notify you when the peer has connected/disconnected
		peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			fmt.Printf("Connection State has changed %s \n", connectionState.String())

			if connectionState == webrtc.ICEConnectionStateConnected {
				fmt.Println("ICE connection was successful")
			} else if connectionState == webrtc.ICEConnectionStateFailed ||
				connectionState == webrtc.ICEConnectionStateDisconnected {
				cancel()
			}
		})

		// Wait for context to be done
		<-ctx.Done()
		peerConnection.Close()

	}(peerConnection)

	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	encryptor.Decode(offerStr, &offer)
	// Set the remote SessionDescription
	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	// Create answer
	answer, err = peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	return
}
