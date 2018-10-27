// Copyright 2018 Shift Devices AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/digitalbitbox/bitbox-wallet-app/util/errp"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Trezor models the API of the trezor package.
type Trezor interface {
	Status() string
	PIN(string) error
	Passphrase(string) error
}

// Handlers provides a web API to the Bitbox.
type Handlers struct {
	trezor Trezor
	log    *logrus.Entry
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(
	handleFunc func(string, func(*http.Request) (interface{}, error)) *mux.Route,
	log *logrus.Entry,
) *Handlers {
	handlers := &Handlers{log: log}

	handleFunc("/status", handlers.getStatusHandler).Methods("GET")
	handleFunc("/pin", handlers.postPINHandler).Methods("POST")
	handleFunc("/passphrase", handlers.postPassphraseHandler).Methods("POST")

	return handlers
}

// Init installs a trezor as a base for the web api. This needs to be called before any requests
// are made.
func (handlers *Handlers) Init(trezor Trezor) {
	handlers.log.Debug("Init")
	handlers.trezor = trezor
}

// Uninit removes the bitbox. After this, not requests should be made.
func (handlers *Handlers) Uninit() {
	handlers.log.Debug("Uninit")
	handlers.trezor = nil
}
func (handlers *Handlers) getStatusHandler(_ *http.Request) (interface{}, error) {
	return handlers.trezor.Status(), nil
}

func (handlers *Handlers) postPINHandler(r *http.Request) (interface{}, error) {
	var pin string
	if err := json.NewDecoder(r.Body).Decode(&pin); err != nil {
		return nil, errp.WithStack(err)
	}
	return nil, handlers.trezor.PIN(pin)
}

func (handlers *Handlers) postPassphraseHandler(r *http.Request) (interface{}, error) {
	var passphrase string
	if err := json.NewDecoder(r.Body).Decode(&passphrase); err != nil {
		return nil, errp.WithStack(err)
	}
	return nil, handlers.trezor.Passphrase(passphrase)
}
