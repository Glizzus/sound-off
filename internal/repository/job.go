package repository

type UpcomingJobQueue interface {
	Pull()
}
