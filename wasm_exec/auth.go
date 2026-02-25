package main

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"sync"
)

const TokenHeader = "X-Exec-Secret"

var ErrUnauthorized = errors.New("not authorized")

// Authorizer help secure our same-process localhost web server communication.
// Tokens are unique per request and only valid once.
type Authorizer[H hash.Hash] struct {
	mtx          sync.Mutex
	sequences    map[int][]byte
	hash         func() H
	keyLen       int
	secret       string
	nextSequence int
}

type Token struct {
	sequence int
	salt     [4]byte
	key      []byte
}

func NewAuthorizer[H hash.Hash](h func() H) *Authorizer[H] {
	a := &Authorizer[H]{
		sequences: make(map[int][]byte),
		hash:      h,
		keyLen:    h().Size(),
	}

	a.secret = rand.Text()
	return a
}

func (a *Authorizer[H]) Issue() (t *Token, err error) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	a.nextSequence++

	t = &Token{
		sequence: a.nextSequence,
	}
	rand.Read(t.salt[:])

	t.key, err = pbkdf2.Key(a.hash, a.secret, t.salt[:], t.sequence, a.keyLen)
	if err != nil {
		return nil, err
	}

	a.sequences[t.sequence] = t.key
	return t, nil
}

func (a *Authorizer[H]) Verify(t *Token) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	key, err := pbkdf2.Key(a.hash, a.secret, t.salt[:], t.sequence, a.keyLen)
	if err != nil {
		return err
	}

	if !bytes.Equal(key, a.sequences[t.sequence]) {
		return ErrUnauthorized
	}

	delete(a.sequences, t.sequence)
	return nil
}

func (a *Authorizer[H]) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t, err := TokenFromString(r.Header.Get(TokenHeader))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := a.Verify(t); err != nil {
			if errors.Is(err, ErrUnauthorized) {
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			fmt.Fprintf(w, "%v\r\n", err)
			return
		}
		r.Header.Del(TokenHeader)
		next.ServeHTTP(w, r)
	})
}

type encodedToken struct {
	Sequence int    `json:"sequence"`
	Salt     string `json:"salt"`
	Key      string `json:"key"`
}

func (t *Token) String() string {
	et := &encodedToken{
		Sequence: t.sequence,
		Salt:     base64.StdEncoding.EncodeToString(t.salt[:]),
		Key:      base64.StdEncoding.EncodeToString(t.key),
	}

	jt, err := json.Marshal(et)
	if err != nil {
		panic(err) // impossible
	}

	return base64.StdEncoding.EncodeToString(jt)
}

func TokenFromString(s string) (t *Token, err error) {
	et := &encodedToken{}
	jt, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jt, et); err != nil {
		return nil, err
	}
	t = &Token{
		sequence: et.Sequence,
	}
	if salt, err := base64.StdEncoding.DecodeString(et.Salt); err != nil {
		return nil, err
	} else {
		copy(t.salt[:], salt)
	}
	if t.key, err = base64.StdEncoding.DecodeString(et.Key); err != nil {
		return nil, err
	}
	return t, nil
}
