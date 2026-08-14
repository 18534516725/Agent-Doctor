package proxy

import (
	"net/http"
	"strings"
)

var forwardedRequestHeaders = map[string]struct{}{
	"accept": {}, "content-type": {}, "authorization": {}, "x-api-key": {},
	"anthropic-version": {}, "anthropic-beta": {}, "openai-organization": {},
	"openai-project": {}, "user-agent": {},
}

var hopByHopHeaders = map[string]struct{}{
	"connection": {}, "proxy-connection": {}, "keep-alive": {}, "proxy-authenticate": {},
	"proxy-authorization": {}, "te": {}, "trailer": {}, "transfer-encoding": {}, "upgrade": {},
}

func copyForwardHeaders(destination, source http.Header) {
	for name, values := range source {
		if _, ok := forwardedRequestHeaders[strings.ToLower(name)]; !ok {
			continue
		}
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}

func copyResponseHeaders(destination, source http.Header) {
	for name, values := range source {
		if _, blocked := hopByHopHeaders[strings.ToLower(name)]; blocked {
			continue
		}
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}
