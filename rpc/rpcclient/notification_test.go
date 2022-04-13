// Copyright 2018 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"gitlab.com/aquachain/aquachain/rpc"
)

type NotificationTestService struct {
	mu           sync.Mutex
	unsubscribed bool

	gotHangSubscriptionReq  chan struct{}
	unblockHangSubscription chan struct{}
}

func (s *NotificationTestService) Echo(i int) int {
	return i
}

func (s *NotificationTestService) wasUnsubCallbackCalled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unsubscribed
}

func (s *NotificationTestService) Unsubscribe(subid string) {
	s.mu.Lock()
	s.unsubscribed = true
	s.mu.Unlock()
}

func (s *NotificationTestService) SomeSubscription(ctx context.Context, n, val int) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, ErrNotificationsUnsupported
	}

	// by explicitly creating an subscription we make sure that the subscription id is send back to the client
	// before the first subscription.Notify is called. Otherwise the events might be send before the response
	// for the aqua_subscribe method.
	subscription := notifier.CreateSubscription()

	go func() {
		// test expects n events, if we begin sending event immediately some events
		// will probably be dropped since the subscription ID might not be send to
		// the client.
		time.Sleep(5 * time.Second)
		for i := 0; i < n; i++ {
			if err := notifier.Notify(subscription.ID, val+i); err != nil {
				return
			}
		}

		select {
		case <-notifier.Closed():
			s.mu.Lock()
			s.unsubscribed = true
			s.mu.Unlock()
		case <-subscription.Err():
			s.mu.Lock()
			s.unsubscribed = true
			s.mu.Unlock()
		}
	}()

	return subscription, nil
}

// HangSubscription blocks on s.unblockHangSubscription before
// sending anything.
func (s *NotificationTestService) HangSubscription(ctx context.Context, val int) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, ErrNotificationsUnsupported
	}

	s.gotHangSubscriptionReq <- struct{}{}
	<-s.unblockHangSubscription
	subscription := notifier.CreateSubscription()

	go func() {
		notifier.Notify(subscription.ID, val)
	}()
	return subscription, nil
}

func waitForMessages(t *testing.T, in *json.Decoder, successes chan<- jsonSuccessResponse,
	failures chan<- jsonErrResponse, notifications chan<- jsonNotification, errors chan<- error) {

	// read and parse server messages
	for {
		var rmsg json.RawMessage
		if err := in.Decode(&rmsg); err != nil {
			return
		}

		var responses []map[string]interface{}
		if rmsg[0] == '[' {
			if err := json.Unmarshal(rmsg, &responses); err != nil {
				errors <- fmt.Errorf("Received invalid message: %s", rmsg)
				return
			}
		} else {
			var msg map[string]interface{}
			if err := json.Unmarshal(rmsg, &msg); err != nil {
				errors <- fmt.Errorf("Received invalid message: %s", rmsg)
				return
			}
			responses = append(responses, msg)
		}

		for _, msg := range responses {
			// determine what kind of msg was received and broadcast
			// it to over the corresponding channel
			if _, found := msg["result"]; found {
				successes <- jsonSuccessResponse{
					Version: msg["jsonrpc"].(string),
					Id:      msg["id"],
					Result:  msg["result"],
				}
				continue
			}
			if _, found := msg["error"]; found {
				params := msg["params"].(map[string]interface{})
				failures <- jsonErrResponse{
					Version: msg["jsonrpc"].(string),
					Id:      msg["id"],
					Error:   rpc.JsonError{int(params["subscription"].(float64)), params["message"].(string), params["data"]},
				}
				continue
			}
			if _, found := msg["params"]; found {
				params := msg["params"].(map[string]interface{})
				notifications <- jsonNotification{
					Version: msg["jsonrpc"].(string),
					Method:  msg["method"].(string),
					Params:  jsonSubscription{params["subscription"].(string), params["result"]},
				}
				continue
			}
			errors <- fmt.Errorf("Received invalid message: %s", msg)
		}
	}
}
