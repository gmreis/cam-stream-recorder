package stream

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts"
)

// mpegtsMuxer allows to save a H264 / MPEG-4 audio stream into a MPEG-TS file, rotacionando arquivos por tamanho e nomeando com timestamps.
type mpegtsMuxer struct {
	fileNameBase string
	notify       chan<- string // channel to notify when a file is ready
	h264Format   *format.H264

	f         *os.File
	b         *bufio.Writer
	w         *mpegts.Writer
	h264Track *mpegts.Track

	dtsExtractor *h264.DTSExtractor
	mutex        sync.Mutex

	maxBytes int64
	written  int64

	startPTS  int64
	endPTS    int64
	startTime time.Time
	endTime   time.Time
	clockRate int
	fileIndex int
}

func NewMpegtsMuxer(fileNameBase string, h264Format *format.H264, notify chan<- string) *mpegtsMuxer {
	mpegtsMuxer := &mpegtsMuxer{
		fileNameBase: fileNameBase,
		notify:       notify,
		h264Format:   h264Format,
		dtsExtractor: nil,
		mutex:        sync.Mutex{},

		maxBytes:  10 * 1024 * 1024, // 1 MB
		fileIndex: 1,
		clockRate: 90000, // for H264
	}

	mpegtsMuxer.openNewFile() // Open the first file

	return mpegtsMuxer
}

// func (e *mpegtsMuxer) initialize() error {
// 	e.maxBytes = 1 * 1024 * 1024 // 1 MB
// 	e.fileIndex = 1
// 	e.clockRate = 90000 // para H264
// 	return e.openNewFile()
// }

func (e *mpegtsMuxer) openNewFile() error {
	if e.f != nil {
		e.close() // fecha o arquivo anterior, se existir
	}
	fileName := fmt.Sprintf("%s_tmp.ts", e.fileNameBase)
	fmt.Printf("Opening new file: %s\n", fileName)
	var err error
	e.f, err = os.Create(fileName)
	if err != nil {
		return err
	}
	e.b = bufio.NewWriter(e.f)

	e.h264Track = &mpegts.Track{
		Codec: &mpegts.CodecH264{},
	}
	e.w = &mpegts.Writer{W: e.b, Tracks: []*mpegts.Track{e.h264Track}}
	err = e.w.Initialize()
	if err != nil {
		return err
	}
	e.written = 0
	e.startPTS = -1
	e.endPTS = -1
	e.startTime = time.Time{}
	e.endTime = time.Time{}
	return nil
}

func (e *mpegtsMuxer) close() {
	e.b.Flush() //nolint:errcheck
	e.f.Close()
	// Renomeia o último arquivo
	if !e.startTime.IsZero() && !e.endTime.IsZero() {
		oldName := fmt.Sprintf("%s_tmp.ts", e.fileNameBase)
		newName := fmt.Sprintf("%s_%s_%s.ts", e.fileNameBase,
			e.startTime.Format("20060102150405"),
			e.endTime.Format("20060102150405"))
		os.Rename(oldName, newName)

		fmt.Println("File renamed to:", newName)
		// enviar o arquivo para o S3 usando go rotine / channels
		// e.notify <- newName
		fmt.Println("File notification sent:", newName)
	}

	fmt.Println("Closed MPEG-TS muxer, file saved:", e.fileNameBase)
}

// ptsToTime converte o PTS para time.Time relativo ao início do arquivo
func (e *mpegtsMuxer) ptsToTime(pts int64) time.Time {
	if e.startPTS < 0 {
		e.startPTS = pts
		e.startTime = time.Now()
	}
	sec := float64(pts-e.startPTS) / float64(e.clockRate)
	return e.startTime.Add(time.Duration(sec * float64(time.Second)))
}

// writeH264 writes a H264 access unit into MPEG-TS, rotacionando arquivo no próximo IDR após atingir o limite.
func (e *mpegtsMuxer) writeH264(au [][]byte, pts int64) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var filteredAU [][]byte //nolint:prealloc

	nonIDRPresent := false
	idrPresent := false

	for _, nalu := range au {
		typ := h264.NALUType(nalu[0] & 0x1F)
		switch typ {
		case h264.NALUTypeSPS:
			e.h264Format.SPS = nalu
			continue

		case h264.NALUTypePPS:
			e.h264Format.PPS = nalu
			continue

		case h264.NALUTypeAccessUnitDelimiter:
			continue

		case h264.NALUTypeIDR:
			idrPresent = true

		case h264.NALUTypeNonIDR:
			nonIDRPresent = true
		}

		filteredAU = append(filteredAU, nalu)
	}

	au = filteredAU

	if au == nil || (!nonIDRPresent && !idrPresent) {
		return nil
	}

	// add SPS and PPS before access unit that contains an IDR
	if idrPresent {
		au = append([][]byte{e.h264Format.SPS, e.h264Format.PPS}, au...)
	}

	if e.dtsExtractor == nil {
		// skip samples silently until we find one with a IDR
		if !idrPresent {
			return nil
		}
		e.dtsExtractor = &h264.DTSExtractor{}
		e.dtsExtractor.Initialize()
	}

	dts, err := e.dtsExtractor.Extract(au, pts)
	if err != nil {
		return err
	}

	// Atualiza timestamps para nomeação
	currentTime := e.ptsToTime(pts)
	if e.startTime.IsZero() {
		e.startTime = currentTime
	}
	e.endTime = currentTime
	e.endPTS = pts

	err = e.w.WriteH264(e.h264Track, pts, dts, au)
	if err != nil {
		return fmt.Errorf("failed to write H264 to MPEG-TS: %w", err)
	}

	totalSize := 0
	for _, nalu := range au {
		e.written += int64(len(nalu))
	}
	fmt.Printf("Wrote H264 access unit with PTS %d, DTS %d, size %d, %d bytes\n", pts, dts, len(au), totalSize)

	e.written += int64(totalSize)

	// ROTATION LOGIC: only rotation if IDR is present and the limit is reached
	if idrPresent && e.written >= e.maxBytes {
		fmt.Println("Max bytes reached, rotating file...")
		if err := e.openNewFile(); err != nil {
			return err
		}
	}

	return nil
}
