// Package opus handles encoding, decoding, and streaming of Opus audio frames
// for Discord voice playback.
//
// Audio is stored in a minimal binary format: concatenated length-prefixed frames
// ([uint16 LE length][opus bytes]). No headers, no metadata.
//
// Encode transcodes any audio to Opus via FFmpeg and produces length-prefixed frames.
// Decode reads length-prefixed frames back. Stream sends decoded frames to a
// Discord voice connection.
package opus
