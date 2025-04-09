package handler

import "fmt"

// SoundCronAlreadyExistsError is an error that indicates
// that a soundcron already exists for the given guild and name.
type SoundCronAlreadyExistsError struct {
	GuildID string
	Name    string
}

func (e *SoundCronAlreadyExistsError) Error() string {
	return fmt.Sprintf("soundcron already exists for guild %s with name %s", e.GuildID, e.Name)
}

var _ error = (*SoundCronAlreadyExistsError)(nil)

// UserError is an error type that is used to represent
// an error that should be displayed to the user.
type UserError struct {
	Message string
}

func (e *UserError) Error() string {
	return e.Message
}

var _ error = (*UserError)(nil)
