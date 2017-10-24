package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// SubscriberHandler handles requests from clients. It has info on subscribers
// and currency
type SubscriberHandler struct {
	db      SubscriberDB
	monitor CurrencyMonitor
}

// SubscriberHandlerFactory returns a fresh handler
func SubscriberHandlerFactory(db SubscriberDB, monitor CurrencyMonitor) SubscriberHandler {
	handler := SubscriberHandler{db: db, monitor: monitor}

	return handler
}

func (handler *SubscriberHandler) handleSubscriberRequestPOST(res http.ResponseWriter, req *http.Request) {

	// attempt to decode the POST json
	var s Subscriber
	err := json.NewDecoder(req.Body).Decode(&s)

	// if couldn't decode -> bad req
	// (SHOULD ALSO FAIL FOR NON-COMPLIANT JSON)
	if err != nil {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// check validity of posted json
	if !validateSubscriber(s) {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// check validity of URL in posted json
	_, err = url.ParseRequestURI(*s.WebhookURL)
	if err != nil {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// (try to) add the student
	id, addErr := handler.db.Add(s)

	// if couldn't add -> internal server error
	//  (client's responsability to retry)
	if addErr != nil {
		respWithCode(&res, http.StatusInternalServerError)
		return
	}

	// respond with id given by db
	fmt.Fprint(res, id)
}

func (handler *SubscriberHandler) handleSubscriberRequestGET(res http.ResponseWriter, req *http.Request) {

	// try to pick out the id from the url
	parts := strings.Split(req.URL.String(), "/")
	if len(parts) < 2 {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// convert (string) id to int
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// attempt to fetch subscriber with given id
	sub, err := handler.db.Get(id)
	if err != nil {
		respWithCode(&res, http.StatusNotFound)
		return
	}

	http.Header.Add(res.Header(), "content-type", "application/json")

	// decode and send the sub
	err = json.NewEncoder(res).Encode(sub)
	if err != nil {
		respWithCode(&res, http.StatusInternalServerError)
		return
	}

}

func (handler *SubscriberHandler) handleSubscriberRequestDELETE(res http.ResponseWriter, req *http.Request) {

	// try to pick out the id from the url
	parts := strings.Split(req.URL.String(), "/")
	if len(parts) < 2 {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// convert (string) id to int
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// attempt to delete the subscriber with id
	err = handler.db.Remove(id)
	if err != nil {
		respWithCode(&res, http.StatusNotFound)
		return
	}

	// deletion succeeded. yay!
	respWithCode(&res, http.StatusOK)
}

// handle subscriber requests
func (handler *SubscriberHandler) handleSubscriberRequest(res http.ResponseWriter, req *http.Request) {

	// switch on the method of the request
	switch req.Method {
	case "POST":
		handler.handleSubscriberRequestPOST(res, req)
	case "GET":
		handler.handleSubscriberRequestGET(res, req)
	case "DELETE":
		handler.handleSubscriberRequestDELETE(res, req)
	default:
		respWithCode(&res, http.StatusNotImplemented)
	}
}

// handle requests about latests data
func (handler *SubscriberHandler) handleLatest(res http.ResponseWriter, req *http.Request) {

	// ..only supports POST method
	if req.Method != http.MethodPost {
		respWithCode(&res, http.StatusNotImplemented)
	}

	// attempt to decode the POST json
	var currReq CurrencyRequest
	err := json.NewDecoder(req.Body).Decode(&currReq)

	// if couldn't decode -> bad req
	if err != nil {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// check validity of posted json
	if !validateCurrencyRequest(currReq) {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// (try to) get the latest currency info
	rate, rateErr := handler.monitor.Latest(*currReq.BaseCurrency, *currReq.TargetCurrency)

	// if couldn't get latest -> either not found or internal error
	//  (client's responsability to retry)
	if rateErr == errInvalidCurrency {
		respWithCode(&res, http.StatusBadRequest)
		return
	} else if rateErr != nil {
		respWithCode(&res, http.StatusInternalServerError)
		return
	}

	// respond with id given by db
	fmt.Fprint(res, rate)
}

// handle requests about average data
func (handler *SubscriberHandler) handleAverage(res http.ResponseWriter, req *http.Request) {

	// ..only supports POST method
	if req.Method != http.MethodPost {
		respWithCode(&res, http.StatusNotImplemented)
	}

	// attempt to decode the POST json
	var currReq CurrencyRequest
	err := json.NewDecoder(req.Body).Decode(&currReq)

	// if couldn't decode -> bad req
	if err != nil {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// check validity of posted json
	if !validateCurrencyRequest(currReq) {
		respWithCode(&res, http.StatusBadRequest)
		return
	}

	// (try to) get the average currency info for the last 7 days
	rate, rateErr := handler.monitor.Average(*currReq.BaseCurrency, *currReq.TargetCurrency, 7)

	// if couldn't get average -> either not found or internal error
	//  (client's responsability to retry)
	if rateErr == errInvalidCurrency {
		respWithCode(&res, http.StatusBadRequest)
		return
	} else if rateErr != nil {
		respWithCode(&res, http.StatusInternalServerError)
		return
	}

	// respond with id given by db
	fmt.Fprint(res, rate)
}

// handler (for testing and debug mostly) that forces all subscribers to be notfied
func (handler *SubscriberHandler) handleEvaluationTrigger(res http.ResponseWriter, req *http.Request) {

	// only GET supported
	if req.Method != http.MethodGet {
		respWithCode(&res, http.StatusNotImplemented)
		return
	}

	// notify all subscribers
	err := handler.notifyAll()

	if err != nil {
		respWithCode(&res, http.StatusInternalServerError)
		return
	}
}

// notify all subscribers
func (handler *SubscriberHandler) notifyAll() error {
	subs, err := handler.db.GetAll()
	if err != nil {
		return err
	}
	for _, s := range subs {
		handler.notifySubscriber(s)
	}
	return nil
}

// notify single subscriber
func (handler *SubscriberHandler) notifySubscriber(s Subscriber) {
	// TODO implement notifications
	fmt.Println("Notifying ", *s.WebhookURL)
}

// utility function for responding with a simple statuscode
func respWithCode(res *http.ResponseWriter, status int) {
	http.Error(*res, http.StatusText(status), status)
}
