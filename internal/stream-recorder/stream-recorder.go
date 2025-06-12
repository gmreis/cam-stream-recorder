package stream

import (
	"errors"
	"fmt"
	"log"

	"github.com/gmreis/cam-stream-recorder/internal/config"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/pion/rtp"
)

type StreamRecorder struct {
	camera           config.CameraConfig
	localStoragePath string
	client           *gortsplib.Client
}

func NewStreamRecorder(camera config.CameraConfig, localStoragePath string) *StreamRecorder {
	return &StreamRecorder{
		camera:           camera,
		localStoragePath: localStoragePath,
		client:           &gortsplib.Client{},
	}
}

func (sr *StreamRecorder) Initialize() error {
	u, err := base.ParseURL(sr.camera.RTSP)
	if err != nil {
		panic(err)
	}
	fmt.Println("Parsed URL:", u)

	// connect to the server
	err = sr.client.Start(u.Scheme, u.Host)
	if err != nil {
		panic(err)
	}
	// defer sr.client.Close()

	// find available medias
	desc, _, err := sr.client.Describe(u)
	if err != nil {
		panic(err)
	}

	log.Printf("reading %d medias", len(desc.Medias))
	for i, media := range desc.Medias {
		log.Printf("media %d: %s", i+1, media.Marshal())
		for j, f := range media.Formats {
			log.Printf("\rformat %d: %s - %d", j+1, f.Codec(), f.PayloadType())
		}
	}

	// find the H264 media and format
	var h264Format *format.H264
	h264Media := desc.FindFormat(&h264Format)
	if h264Media == nil {
		panic("H264 media not found")
	}

	// setup RTP -> H264 decoder
	h264RTPDec, err := h264Format.CreateDecoder()
	if err != nil {
		panic(err)
	}

	// setup MPEG-TS muxer
	mpegtsMuxer := NewMpegtsMuxer("mystream2", h264Format, nil)

	// setup all medias
	err = sr.client.SetupAll(desc.BaseURL, desc.Medias)
	if err != nil {
		panic(err)
	}

	// called when a RTP packet arrives
	sr.client.OnPacketRTP(h264Media, h264Format, func(pkt *rtp.Packet) {
		// decode timestamp
		pts, ok := sr.client.PacketPTS2(h264Media, pkt)
		if !ok {
			log.Printf("h264: waiting for timestamp")
			return
		}

		// extract access unit from RTP packets
		au, err := h264RTPDec.Decode(pkt)
		if err != nil {
			if !errors.Is(err, rtph264.ErrNonStartingPacketAndNoPrevious) && !errors.Is(err, rtph264.ErrMorePacketsNeeded) {
				log.Printf("fail to decode packet to h264: %v", err)
			}
			return
		}

		// fmt.Println("Decoded H264 access unit:", au, pts)

		// encode the access unit into MPEG-TS
		err = mpegtsMuxer.writeH264(au, pts)
		if err != nil {
			log.Printf("fail to write H264 to MPEG-TS: %v", err)
			return
		}
		log.Printf("h264 saved TS packet")
	})

	return nil
}

func (sr *StreamRecorder) StartRecording() error {
	// start playing
	_, err := sr.client.Play(nil)
	if err != nil {
		panic(err)
	}

	// wait until a fatal error
	fmt.Println("Recording started, waiting for stream...")
	err = sr.client.Wait()
	if err != nil && err.Error() != "EOF" {
		fmt.Printf("Recording stopped with error: %v\n", err)
		panic(err)
	}

	fmt.Println("Recording stopped successfully")

	return nil
}

func (sr *StreamRecorder) StopRecording() {
	if sr.client != nil {
		sr.client.Close()
	}
	log.Println("Recording stopped")
}

// func StreamRecorder(camera CameraConfig, localProvider LocalProvider) error {
// 	// Initialize the stream recorder with the provided stream URL
// 	recorder := NewRecorder(camera.rtsp)
// 	if recorder == nil {
// 		return fmt.Errorf("failed to initialize recorder for stream: %s", camera.rtsp)
// 	}

// 	// Start recording the stream
// 	if err := recorder.Start(); err != nil {
// 		return fmt.Errorf("failed to start recording: %w", err)
// 	}

// 	if err := recorder.SaveToProvider(localProvider); err != nil {
// 		return fmt.Errorf("failed to save recording to provider %s: %w", localProvider.GetProviderName(), err)
// 	}

// 	// Process each storage provider
// 	// for _, providerName := range storageProviders {
// 	// 	provider, err := GetStorageProvider(providerName)
// 	// 	if err != nil {
// 	// 		return fmt.Errorf("failed to get storage provider %s: %w", providerName, err)
// 	// 	}

// 	// 	if err := recorder.SaveToProvider(provider); err != nil {
// 	// 		return fmt.Errorf("failed to save recording to provider %s: %w", provider.GetProviderName(), err)
// 	// 	}
// 	// }

// 	return nil
// }
