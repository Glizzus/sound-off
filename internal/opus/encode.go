package opus

import (
	"encoding/binary"
	"errors"
	"io"
	"os/exec"

	"github.com/jonas747/ogg"
)

// TranscodeFunc takes raw audio and returns a stream of OGG/Opus data.
// The returned ReadCloser must be closed to release any underlying resources.
type TranscodeFunc func(r io.Reader) (io.ReadCloser, error)

// FFmpegTranscode is the default TranscodeFunc that shells out to ffmpeg.
func FFmpegTranscode(r io.Reader) (io.ReadCloser, error) {
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:0",
		"-vn",
		"-map", "0:a",
		"-acodec", "libopus",
		"-f", "ogg",
		"-vbr", "on",
		"-compression_level", "5",
		"-ar", "48000",
		"-ac", "2",
		"-b:a", "64000",
		"-application", "audio",
		"-frame_duration", "20",
		"-packet_loss", "1",
		"-threads", "0",
		"pipe:1",
	)

	cmd.Stdin = r

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &cmdReadCloser{ReadCloser: stdout, cmd: cmd}, nil
}

// Encoder transcodes audio to length-prefixed Opus frames. The transcoding
// step is provided via a TranscodeFunc, keeping the OGG demux → frames
// pipeline generic.
type Encoder struct {
	Transcode TranscodeFunc
}

// NewEncoder returns an Encoder that uses the given TranscodeFunc.
func NewEncoder(fn TranscodeFunc) *Encoder {
	return &Encoder{Transcode: fn}
}

// Encode reads audio from r, transcodes it to OGG/Opus via the configured
// TranscodeFunc, then demuxes the OGG stream into length-prefixed Opus frames.
// The returned io.ReadCloser must be closed to clean up resources.
func (e *Encoder) Encode(r io.Reader) (io.ReadCloser, error) {
	oggStream, err := e.Transcode(r)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		defer oggStream.Close()

		decoder := ogg.NewPacketDecoder(ogg.NewDecoder(oggStream))

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

	return pr, nil
}

// Encode is a convenience function that transcodes using FFmpeg.
func Encode(r io.Reader) (io.ReadCloser, error) {
	return NewEncoder(FFmpegTranscode).Encode(r)
}

// cmdReadCloser wraps a command's stdout and kills the process on Close.
type cmdReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c *cmdReadCloser) Close() error {
	err := c.ReadCloser.Close()
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	c.cmd.Wait()
	return err
}
