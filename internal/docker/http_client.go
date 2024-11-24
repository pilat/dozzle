package docker

import (
	"bufio"
	"io"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
)

type dockerClientTransport struct {
	transport http.RoundTripper
}

// newDockerHTTPClient creates a new HTTP client the same way as Docker Client does internally. It adds a RoundTripper
// to the HTTP client that modifies the health status of the events stream to make docker client compatible with podman.
func newDockerHTTPClient() (*http.Client, error) {
	hostURL, err := client.ParseHostURL(client.DefaultDockerHost)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{}
	err = sockets.ConfigureTransport(transport, hostURL.Scheme, hostURL.Host)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport:     &dockerClientTransport{transport},
		CheckRedirect: client.CheckRedirect,
	}, nil
}

func (rt *dockerClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/events") {
		originalBodyReader := resp.Body
		bodyReader, bodyWriter := io.Pipe()
		resp.Body = bodyReader

		go func() {
			scanner := bufio.NewScanner(originalBodyReader)
			for scanner.Scan() {
				msg := scanner.Bytes()
				modifiedMsg := updateHealthAction(msg)
				bodyWriter.Write(append(modifiedMsg, '\n'))
			}

			originalBodyReader.Close()
			bodyWriter.Close()
		}()
	}

	return resp, nil
}

func updateHealthAction(msg []byte) []byte {
	action, _ := jsonparser.GetString(msg, "Action")
	if action != "health_status" {
		return msg
	}

	healthStatus, _ := jsonparser.GetString(msg, "HealthStatus")

	switch string(healthStatus) {
	case "starting":
		action = "health_status: running"
	case "unhealthy":
		action = "health_status: unhealthy"
	case "healthy":
		action = "health_status: healthy"
	default:
		return msg
	}

	newMsg, err := jsonparser.Set(msg, []byte(`"`+action+`"`), "Action")
	if err != nil {
		return msg
	}

	return newMsg
}
