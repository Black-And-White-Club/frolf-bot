package round

import "net/http"

// RoundHandler defines the interface for handling HTTP requests for rounds.
type RoundHandler interface {
	GetRounds(w http.ResponseWriter, r *http.Request)
	GetRound(w http.ResponseWriter, r *http.Request)
	ScheduleRound(w http.ResponseWriter, r *http.Request)
	JoinRound(w http.ResponseWriter, r *http.Request)
	EditRound(w http.ResponseWriter, r *http.Request)
	DeleteRound(w http.ResponseWriter, r *http.Request)
	SubmitScore(w http.ResponseWriter, r *http.Request)
	FinalizeRound(w http.ResponseWriter, r *http.Request)
}
