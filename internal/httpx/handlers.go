package httpx

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/you/go-jobsity-flights/internal/providers"
	"github.com/you/go-jobsity-flights/internal/service"
)

type searchResponse struct {
	Origin      string                  `json:"origin"`
	Destination string                  `json:"destination"`
	Date        string                  `json:"date"`
	Cheapest    providers.FlightOffer   `json:"cheapest"`
	Fastest     providers.FlightOffer   `json:"fastest"`
	Offers      []providers.FlightOffer `json:"offers"`
}

func SearchHandler(svc *service.SearchService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		origin := strings.ToUpper(q.Get("origin"))
		dest := strings.ToUpper(q.Get("destination"))
		date := q.Get("date")
		if origin == "" || dest == "" || date == "" {
			http.Error(w, "origin, destination and date are required", http.StatusBadRequest)
			return
		}
		res, err := svc.Search(r.Context(), origin, dest, date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(searchResponse{
			Origin: origin, Destination: dest, Date: date,
			Cheapest: res.Cheapest, Fastest: res.Fastest, Offers: res.All,
		})
	}
}

func HistoryHandler(hist *service.HistoryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		origin := strings.ToUpper(q.Get("origin"))
		dest := strings.ToUpper(q.Get("destination"))
		if origin == "" || dest == "" {
			http.Error(w, "origin and destination are required", http.StatusBadRequest)
			return
		}
		series := hist.MonthlyAverages(origin, dest, 24)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(series)
	}
}

func SubscribeSSEHandler(svc *service.SearchService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sse/"), "/")
		if len(parts) < 2 {
			http.Error(w, "use /sse/{origin}/{destination}?date=YYYY-MM-DD", 400)
			return
		}
		origin := strings.ToUpper(parts[0])
		dest := strings.ToUpper(parts[1])
		date := r.URL.Query().Get("date")
		if date == "" {
			http.Error(w, "date required", 400)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}

		updateTick := time.NewTicker(30 * time.Second)
		defer updateTick.Stop()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				log.Println("SSE client closed")
				return

			case <-updateTick.C:
				res, err := svc.Search(ctx, origin, dest, date)
				if err != nil {
					fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
					flusher.Flush()
					// decide if you want to continue or end; returning ends the stream
					return
				}
				payload, _ := json.Marshal(res)
				fmt.Fprintf(w, "event: update\ndata: %s\n\n", payload)
				flusher.Flush()
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 🔒 in prod, restrict origin
	},
}

func SubscribeWSHandler(svc *service.SearchService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
		if len(parts) < 2 {
			http.Error(w, "use /ws/{origin}/{destination}?date=YYYY-MM-DD", 400)
			return
		}
		origin := strings.ToUpper(parts[0])
		dest := strings.ToUpper(parts[1])
		date := r.URL.Query().Get("date")
		if date == "" {
			http.Error(w, "date required", 400)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		ctx := r.Context()
		for {
			res, err := svc.Search(ctx, origin, dest, date)
			if err != nil {
				conn.WriteJSON(map[string]string{"error": err.Error()})
				return
			}
			if err := conn.WriteJSON(res); err != nil {
				log.Printf("write error: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				continue
			}
		}
	}
}
