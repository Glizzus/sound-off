package opus

import (
	"encoding/binary"
	"errors"
	"io"
	"os/exec"

	"github.com/jonas747/ogg"
)

// Encode takes any audio as an io.Reader, runs FFmpeg to transcode it to Opus,
// and returns an io.Reader that produces length-prefixed Opus frames.
// The caller should read until EOF. The returned io.ReadCloser must be closed
// to clean up the FFmpeg process.
func Encode(r io.Reader) (io.ReadCloser, error) {
	ffmpeg := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-vn",
		"-map", "0:a",
		"-acodec", "libopus",
		"-f", "ogg",
		"-vbr", "on",
		"-compression_level", "10",
		"-ar", "48000",
		"-ac", "2",
		"-b:a", "64000",
		"-application", "audio",
		"-frame_duration", "20",
		"-packet_loss", "1",
		"-threads", "0",
		"pipe:1",
	)

	ffmpeg.Stdin = r

	stdout, err := ffmpeg.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := ffmpeg.Start(); err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		defer ffmpeg.Wait()

		decoder := ogg.NewPacketDecoder(ogg.NewDecoder(stdout))

		// Skip the first 2 OGG metadata packets.
		skip := 2
		for {
			packet, _, err := decoder.Decode()
			if skip > 0 {
				skip--
				continue
			}
			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
					pw.CloseWithError(err)
				}
				return
			}

			var lenBuf [2]byte
			binary.LittleEndian.PutUint16(lenBuf[:], uint16(len(packet)))
			if _, err := pw.Write(lenBuf[:]); err != nil {
				return
			}
			if _, err := pw.Write(packet); err != nil {
				return
			}
		}
	}()

	return &encodeCloser{ReadCloser: pr, cmd: ffmpeg}, nil
}

// encodeCloser wraps the pipe reader and ensures the FFmpeg process is cleaned up.
type encodeCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (e *encodeCloser) Close() error {
	err := e.ReadCloser.Close()
	// Kill FFmpeg if still running (e.g. pipe closed early).
	if e.cmd.Process != nil {
		e.cmd.Process.Kill()
	}
	e.cmd.Wait()
	return err
}
