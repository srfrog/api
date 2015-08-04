// This file was automatically generated from
//
//	profile.go
//
// by
//
//	generator -c Result
//
// DO NOT EDIT

package model

import (
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

type Profiles []Profile

type KeyedProfile struct {
	*Profile
	Key *datastore.Key
}

func (ƨ *Profile) Key(key *datastore.Key) *KeyedProfile {
	return &KeyedProfile{
		Profile: ƨ,
		Key:     key,
	}
}

func (ƨ Profiles) Key(keys []*datastore.Key) (keyed []KeyedProfile) {
	if len(keys) != len(ƨ) {
		panic("Key() called on an slice with len(keys) != len(slice)")
	}

	keyed = make([]KeyedProfile, len(ƨ))
	for i := range keyed {
		keyed[i] = KeyedProfile{
			Profile: &ƨ[i],
			Key:     keys[i],
		}
	}
	return
}

// Save will put this Profile into Datastore using the given key.
func (ƨ Profile) Save(ctx context.Context, key ...*datastore.Key) (*datastore.Key, error) {
	if len(key) > 1 {
		panic("zero or one key expected")
	}

	if len(key) == 1 && key[0] != nil {
		return datastore.Put(ctx, key[0], &ƨ)
	}

	return datastore.Put(ctx, datastore.NewIncompleteKey(ctx, "Profile", nil), &ƨ)
}

// SaveWithParent can be used to save this Profile as child of another
// entity.
// This will error if parent == nil.
func (ƨ Profile) SaveWithParent(ctx context.Context, parent *datastore.Key) (*datastore.Key, error) {
	if parent == nil {
		return nil, errors.New("parent key is nil, expected a valid key")
	}
	return datastore.Put(ctx, datastore.NewIncompleteKey(ctx, "Profile", parent), &ƨ)
}

// NewQueryForProfile prepares a datastore.Query that can be
// used to query entities of type Profile.
func NewQueryForProfile() *datastore.Query {
	return datastore.NewQuery("Profile")
}

type ProfileHandler struct{}

func (ƨ ProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	if r.URL.Path == "" {
		var results Profiles
		keys, _ := NewQueryForProfile().GetAll(ctx, &results)
		json.NewEncoder(w).Encode(results.Key(keys))
		return
	}

	k, _ := datastore.DecodeKey(r.URL.Path)
	var entity Profile
	datastore.Get(ctx, k, &entity)
	json.NewEncoder(w).Encode(entity)
}

func ServeProfile(prefix string, muxes ...*http.ServeMux) {
	path := prefix + "Profile" + "/"

	if len(muxes) == 0 {
		http.Handle(path, http.StripPrefix(path, ProfileHandler{}))
	}

	for _, mux := range muxes {
		mux.Handle(path, http.StripPrefix(path, ProfileHandler{}))
	}
}
