package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func StartFFmpeg(ctx context.Context, streamKey string, sdpFile string, outputType string, outputLocation string, outputFormat string ) {
	log.Println("StreamKey: " + streamKey +
		"sdpFile: "+ sdpFile +
		" outputType: " + outputType +
		" outputLocation: " + outputLocation +
		" outputFormat: " + outputFormat)
	targetPath := ""
	ffmpeg := exec.CommandContext(ctx,"");
	log.Println("tada")
	if outputType == "hls" {
		goDir, _ := os.Getwd()
		errDir := os.MkdirAll("vid/"+streamKey, 0755)
		if errDir != nil {
			log.Fatal(errDir)
		}
		targetPath = goDir + "/vid/" + streamKey

		resOptions := []string{"360p", "480p", "720p", "1080p"}

		variants, _ := generateHLSVariant(resOptions, "")
		generatePlaylist(variants, targetPath, "")
		// Create a ffmpeg process that consumes MKV via stdin, and broadcasts out to Stream URL
		//CURRENTLY THIS IS FFMPEG PROCESS RUNS ON CPU NOT ON GPU
		ffmpeg = exec.CommandContext(ctx, "ffmpeg",
			"-protocol_whitelist", "file,udp,rtp", "-i", "rtp-forwarder.sdp", "-map", "0:v:0", "-map", "0:a:0", "-map",
			"0:v:0", "-map", "0:a:0", "-map", "0:v:0", "-map", "0:a:0", "-map", "0:v:0", "-map", "0:a:0", "-c:v", "h264", "-profile:v", "main", "-pix_fmt",
			"yuv420p", "-preset", "faster", "-crf", "20", "-sc_threshold", "0", "-g", "48", "-keyint_min", "48", "-c:a", "aac", "-ac", "2", "-ar", "48000",
			"-filter:v:0", "scale=w=640:h=360:force_original_aspect_ratio=decrease", "-maxrate:v:0", "856k", "-bufsize:v:0", "1200k", "-b:v:0", "800k",
			"-b:a:0", "96k", "-filter:v:1", "scale=w=842:h=480:force_original_aspect_ratio=decrease", "-maxrate:v:1", "1498k", "-bufsize:v:1", "2100k",
			"-b:v:1", "1400k", "-b:a:1", "128k", "-filter:v:2", "scale=w=1280:h=720:force_original_aspect_ratio=decrease", "-maxrate:v:2", "2996k",
			"-bufsize:v:2", "4200k", "-b:v:2", "2800k", "-b:a:2", "128k", "-filter:v:3", "scale=w=1920:h=1080:force_original_aspect_ratio=decrease",
			"-maxrate:v:3", "5350k", "-bufsize:v:3", "7500k", "-b:v:3", "5000k", "-b:a:3", "192k", "-var_stream_map", "v:0,a:0 v:1,a:1 v:2,a:2 v:3,a:3",
			"-hls_time", "4", "-hls_playlist_type", "event",
			"-hls_segment_filename", targetPath+"/"+"video_%v_%03d.ts", targetPath+"/"+"quality_%v.m3u8") //not creating master and different resolution playlist*/
	} else if outputType == "rtmp" {
		ffmpeg = exec.CommandContext(ctx, "ffmpeg", "-protocol_whitelist", "" +
			"file,udp,rtp", "-i", sdpFile,
			"-map", "0:v:0", "-map", "0:a:0",
			//"-map", "0:v:0", "-map", "0:a:0",
			//"-map", "0:v:0", "-map", "0:a:0",
			//"-map", "0:v:0", "-map", "0:a:0",
			"-c:v", "h264", "-profile:v", "main", "-pix_fmt",
			"yuv420p", "-preset", "faster", "-crf", "20", "-sc_threshold", "0", "-g", "48", "-keyint_min", "48",
			"-c:a", "aac", //Audio Coded
			"-ac", "2", //Audio Channels
			"-ar", "48000",
			"-filter:v:0", "scale=w=640:h=360:force_original_aspect_ratio=decrease", "-maxrate:v:0", "856k", "-bufsize:v:0", "1200k", "-b:v:0", "800k", "-b:a:0", "96k",
			//"-filter:v:1", "scale=w=842:h=480:force_original_aspect_ratio=decrease", "-maxrate:v:1", "1498k", "-bufsize:v:1", "2100k", "-b:v:1", "1400k", "-b:a:1", "128k",
			//"-filter:v:2", "scale=w=1280:h=720:force_original_aspect_ratio=decrease", "-maxrate:v:2", "2996k", "-bufsize:v:2", "4200k", "-b:v:2", "2800k", "-b:a:2", "128k",
			//"-filter:v:3", "scale=w=1920:h=1080:force_original_aspect_ratio=decrease", "-maxrate:v:3", "5350k", "-bufsize:v:3", "7500k", "-b:v:3", "5000k", "-b:a:3", "192k",
			//"-var_stream_map", "v:0,a:0 v:1,a:1 v:2,a:2 v:3,a:3",
			"-f", outputFormat, outputLocation)
	} else if outputType == "icecast" {
		ffmpeg = exec.CommandContext(ctx, "ffmpeg",
			"-protocol_whitelist", "file,udp,rtp", "-i", "rtp-forwarder.sdp",  "-c:a", "libmp3lame", "-ac", "1", "-ar", "48000", "-vn",
			"-f", outputFormat,
			outputLocation) //not creating master and different resolution playlist
	}

	ffmpegOut, _ := ffmpeg.StderrPipe()
	if err := ffmpeg.Start(); err != nil {
		panic(err)
	}

	go func() {
		scanner := bufio.NewScanner(ffmpegOut)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
			if ctx.Err() == context.Canceled {
				break
			}
		}
	}()
}
