package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"
)

const (
	// routesAPIURL is the Google Routes API v2 endpoint.
	routesAPIURL = "https://routes.googleapis.com/directions/v2:computeRoutes"

	// googleTimeout is the maximum duration for a Google API call.
	googleTimeout = 5 * time.Second

	// straightLineSpeedMPS is the fallback speed in m/s (~30 km/h, typical urban speed).
	straightLineSpeedMPS = 30.0 / 3.6

	// httpMaxIdleConns is the maximum number of idle (keep-alive) connections
	// kept in the transport pool across all hosts.
	httpMaxIdleConns = 10

	// httpIdleConnTimeout is how long an idle connection is kept in the pool
	// before being closed. 30 s is a safe value for APIs that enforce shorter
	// server-side keep-alive timeouts.
	httpIdleConnTimeout = 30 * time.Second
)

// GoogleRouter implements Router using the Google Routes API v2.
type GoogleRouter struct {
	apiKey     string
	httpClient *http.Client
	// apiURL is the Google Routes API endpoint. Overrideable in tests.
	apiURL string
}

// NewGoogleRouter creates a Router backed by the Google Routes API v2.
// apiKey must be a valid Google Cloud API key with the Routes API enabled.
func NewGoogleRouter(apiKey string) *GoogleRouter {
	transport := &http.Transport{
		MaxIdleConns:        httpMaxIdleConns,
		MaxIdleConnsPerHost: httpMaxIdleConns,
		IdleConnTimeout:     httpIdleConnTimeout,
	}
	return &GoogleRouter{
		apiKey: apiKey,
		apiURL: routesAPIURL,
		httpClient: &http.Client{
			Timeout:   googleTimeout,
			Transport: transport,
		},
	}
}

// Route calls the Google Routes API v2 and returns the primary route.
// On failure it logs the error and falls back to a straight-line estimate
// with IsFallback set to true so callers can detect degraded responses.
func (g *GoogleRouter) Route(ctx context.Context, req RoutingRequest) (*RoutingResponse, error) {
	resp, err := g.callAPI(ctx, req)
	if err != nil {
		log.Printf("routing: google API error (using straight-line fallback): %v", err)
		return straightLineFallback(req), nil
	}
	return resp, nil
}

// callAPI performs the actual HTTP call to the Google Routes API v2.
func (g *GoogleRouter) callAPI(ctx context.Context, req RoutingRequest) (*RoutingResponse, error) {
	// Build request body per Routes API v2 spec.
	body := routesAPIRequest{
		Origin: routesAPIWaypoint{
			Location: routesAPILocation{
				LatLng: routesAPILatLng{
					Latitude:  req.OriginLat,
					Longitude: req.OriginLon,
				},
			},
		},
		Destination: routesAPIWaypoint{
			Location: routesAPILocation{
				LatLng: routesAPILatLng{
					Latitude:  req.DestinationLat,
					Longitude: req.DestinationLon,
				},
			},
		},
		TravelMode:             "DRIVE",
		RoutingPreference:      "TRAFFIC_AWARE",
		ComputeAlternateRoutes: false,
		RouteModifiers: routesAPIRouteModifiers{
			AvoidTolls:    false,
			AvoidHighways: false,
			AvoidFerries:  false,
		},
		LanguageCode: "es-419",
		Units:        "METRIC",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("routing: google: marshal request: %w", err)
	}

	// Apply timeout derived from context or the default google timeout.
	reqCtx, cancel := context.WithTimeout(ctx, googleTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, g.apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("routing: google: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Goog-Api-Key", g.apiKey)
	// Request only the fields we need to minimize response size and latency.
	httpReq.Header.Set("X-Goog-FieldMask", "routes.duration,routes.distanceMeters,routes.polyline.encodedPolyline")

	httpResp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("routing: google: http: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("routing: google: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("routing: google: status %d: %s", httpResp.StatusCode, string(respBytes))
	}

	var apiResp routesAPIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("routing: google: unmarshal response: %w", err)
	}

	if len(apiResp.Routes) == 0 {
		return nil, fmt.Errorf("routing: google: no routes returned")
	}

	route := apiResp.Routes[0]

	// Parse duration string: Google returns e.g. "123s".
	durationS, err := parseDurationSeconds(route.Duration)
	if err != nil {
		return nil, fmt.Errorf("routing: google: parse duration %q: %w", route.Duration, err)
	}

	return &RoutingResponse{
		Polyline:  route.Polyline.EncodedPolyline,
		DistanceM: route.DistanceMeters,
		DurationS: durationS,
	}, nil
}

// straightLineFallback returns a rough estimate based on straight-line distance.
// IsFallback is set to true so callers can detect degraded responses.
// Used when the Google API is unavailable.
func straightLineFallback(req RoutingRequest) *RoutingResponse {
	distM := haversineMeters(req.OriginLat, req.OriginLon, req.DestinationLat, req.DestinationLon)
	durationS := int(float64(distM) / straightLineSpeedMPS)
	// Return an empty polyline on fallback — no encoded path available.
	return &RoutingResponse{
		Polyline:   "",
		DistanceM:  int(distM),
		DurationS:  durationS,
		IsFallback: true,
	}
}

// parseDurationSeconds parses a Google duration string like "123s" into an integer.
func parseDurationSeconds(s string) (int, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty duration string")
	}
	if s[len(s)-1] != 's' {
		return 0, fmt.Errorf("expected duration ending in 's', got %q", s)
	}
	numStr := s[:len(s)-1]
	if len(numStr) == 0 {
		return 0, fmt.Errorf("no number before 's' in %q", s)
	}
	// Ensure every character is a digit (reject floats and other non-integer strings).
	for _, ch := range numStr {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("non-integer duration %q", s)
		}
	}
	var seconds int
	if _, err := fmt.Sscanf(numStr, "%d", &seconds); err != nil {
		return 0, fmt.Errorf("parse %q: %w", s, err)
	}
	return seconds, nil
}

// haversineMeters computes the great-circle distance in meters between two WGS84 points.
func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusM = 6_371_000.0
	const deg2rad = math.Pi / 180.0

	dLat := (lat2 - lat1) * deg2rad
	dLon := (lon2 - lon1) * deg2rad
	lat1r := lat1 * deg2rad
	lat2r := lat2 * deg2rad

	sinDLat := math.Sin(dLat / 2)
	sinDLon := math.Sin(dLon / 2)
	a := sinDLat*sinDLat + math.Cos(lat1r)*math.Cos(lat2r)*sinDLon*sinDLon
	c := 2 * math.Asin(math.Sqrt(a))
	return earthRadiusM * c
}

// --- JSON types for the Google Routes API v2 ---

type routesAPIRequest struct {
	Origin                 routesAPIWaypoint       `json:"origin"`
	Destination            routesAPIWaypoint       `json:"destination"`
	TravelMode             string                  `json:"travelMode"`
	RoutingPreference      string                  `json:"routingPreference"`
	ComputeAlternateRoutes bool                    `json:"computeAlternateRoutes"`
	RouteModifiers         routesAPIRouteModifiers `json:"routeModifiers"`
	LanguageCode           string                  `json:"languageCode"`
	Units                  string                  `json:"units"`
}

type routesAPIWaypoint struct {
	Location routesAPILocation `json:"location"`
}

type routesAPILocation struct {
	LatLng routesAPILatLng `json:"latLng"`
}

type routesAPILatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type routesAPIRouteModifiers struct {
	AvoidTolls    bool `json:"avoidTolls"`
	AvoidHighways bool `json:"avoidHighways"`
	AvoidFerries  bool `json:"avoidFerries"`
}

type routesAPIResponse struct {
	Routes []routesAPIRoute `json:"routes"`
}

type routesAPIRoute struct {
	DistanceMeters int               `json:"distanceMeters"`
	Duration       string            `json:"duration"`
	Polyline       routesAPIPolyline `json:"polyline"`
}

type routesAPIPolyline struct {
	EncodedPolyline string `json:"encodedPolyline"`
}
