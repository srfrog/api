package model

import "google.golang.org/appengine/datastore"

//go:generate generator

// Challenge is an abstract piece of work that can consist
// of many different Tasks.
//
// Saved in Datastore, Challenge will be a child
// entity to Company, so keys pointing to a Challenge
// can be used to obtain the Company that owns it.
type Challenge struct {
	Assignment

	// The tasks that have to be fulfilled in order
	// to successfully complete the Challenge.
	//
	// Result.StartTimes and Result.FinalSubmissions
	// depend on the ordering of this slice. Also it
	// affects the rendering of this Challenge with
	// respect to the user. Therefore it must be
	// guaranteed to be stable.
	// TODO replace with []int64
	Tasks []*datastore.Key `datastore:",noindex",json:",omitempty"`

	// The Resulter to use to compute skills (and
	// therefore a Result) from the outcome of
	// the Submissions to Tasks.
	Resulter int64 `datastore:",noindex",json:",omitempty"`
}
