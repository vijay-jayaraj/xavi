package service

import (
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/xtracdev/xavi/config"
	"github.com/xtracdev/xavi/plugin"
	"github.com/xtracdev/xavi/plugin/timing"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

const (
	aBackendResponse           = "a backend stuff\n"
	bBackendResponse           = "b backend stuff\n"
	bHandlerStuff              = "b stuff\n"
	backendA                   = "backendA"
	backendB                   = "backendB"
	fooURI                     = "/foo"
	multiBackendAdapterFactory = "test-multiroute-plugin"
)

func handleAStuff(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(aBackendResponse))
}

func handleBStuff(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(bBackendResponse))
}

func TestMRConfigListener(t *testing.T) {
	log.SetLevel(log.InfoLevel)

	var bHandler plugin.MultiBackendHandlerFunc = func(m plugin.BackendHandlerMap, w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(bHandlerStuff))

		ah := m[backendA]
		ar := httptest.NewRecorder()
		ah.ServeHTTP(ar, r)
		assert.Equal(t, aBackendResponse, ar.Body.String())

		bh := m[backendB]
		br := httptest.NewRecorder()
		bh.ServeHTTP(br, r)
		assert.Equal(t, bBackendResponse, br.Body.String())
	}

	var BMBAFactory = func(bhMap plugin.BackendHandlerMap) *plugin.MultiBackendAdapter {
		return &plugin.MultiBackendAdapter{
			BackendHandlerCtx: bhMap,
			Handler:           bHandler,
		}
	}

	plugin.RegisterMultiBackendAdapterFactory(multiBackendAdapterFactory, BMBAFactory)

	AServer := httptest.NewServer(http.HandlerFunc(handleAStuff))
	BServer := httptest.NewServer(http.HandlerFunc(handleBStuff))

	defer AServer.Close()
	defer BServer.Close()

	ms := mrtBuildListener(AServer.URL, BServer.URL)

	uriToRoutesMap := ms.organizeRoutesByUri()
	uriToGuardAndHandlerMap := mapRoutesToGuardAndHandler(uriToRoutesMap)
	uriHandlerMap := makeURIHandlerMap(uriToGuardAndHandlerMap)

	assert.Equal(t, 1, len(uriHandlerMap))

	ls := httptest.NewServer(timing.NewTimingWrapper().Wrap(uriHandlerMap[fooURI]))
	defer ls.Close()

	resp, err := http.Get(ls.URL + fooURI)
	assert.Nil(t, err)
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	assert.True(t, strings.Contains(string(body), bHandlerStuff))
}

func makeServerConfig(name string, theURL string) config.ServerConfig {
	parseUrl, _ := url.Parse(theURL)
	host, port, _ := net.SplitHostPort(parseUrl.Host)
	portVal, _ := strconv.Atoi(port)

	println(host)
	println(portVal)

	return config.ServerConfig{
		Name:    name,
		Address: host,
		Port:    portVal,
		PingURI: "/xtracrulesok",
	}
}

func makeBackend(name string, serverConfig config.ServerConfig) *backend {
	servers := []config.ServerConfig{serverConfig}
	var b backend
	b.Name = name
	loadBalancer, err := instantiateLoadBalancer("round-robin", b.Name, "", servers)
	if err != nil {
		panic(err.Error())
	}
	b.LoadBalancer = loadBalancer

	return &b

}

func mrtBuildListener(urlA string, urlB string) *managedService {
	serverA := makeServerConfig("server1", urlA)
	serverB := makeServerConfig("server2", urlB)

	backEndA := makeBackend(backendA, serverA)
	backEndB := makeBackend(backendB, serverB)

	var r1 = route{
		Name:                   "route1",
		URIRoot:                fooURI,
		Backends:               []*backend{backEndA, backEndB},
		MultiBackendPluginName: multiBackendAdapterFactory,
	}

	var ms = managedService{
		Address:      "localhost:23456", //Ignored - we use a testserver with a dyn addr for testing
		ListenerName: "test listener",
		Routes:       []route{r1},
	}

	return &ms

}

func TestMakeGHEntryForSingleBackendRouteProxy(t *testing.T) {
	defer os.Setenv("http_proxy", os.Getenv("http_proxy"))
	defer os.Setenv("https_proxy", os.Getenv("https_proxy"))
	defer os.Setenv("no_proxy", os.Getenv("no_proxy"))
	defer os.Setenv("HTTP_PROXY", os.Getenv("HTTP_PROXY"))
	defer os.Setenv("HTTPS_PROXY", os.Getenv("HTTPS_PROXY"))
	defer os.Setenv("NO_PROXY", os.Getenv("NO_PROXY"))

	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")
	os.Unsetenv("no_proxy")
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("PROXY REACHED"))
	}))
	defer proxyServer.Close()

	os.Setenv("http_proxy", proxyServer.URL)

	server := makeServerConfig("EndpointServer", "http://www.backend.com")
	backEnd := makeBackend(backendA, server)
	testRoute := route{
		Name:     "route1",
		Backends: []*backend{backEnd},
	}

	guardAndHandler := makeGHEntryForSingleBackendRoute(testRoute)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "http://www.input.com", nil)
	assert.Nil(t, err)

	guardAndHandler.HandlerFn.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTeapot, w.Code)
	assert.Equal(t, "PROXY REACHED", w.Body.String())
}

func TestMakeGHEntryForSingleBackendRouteNoProxy(t *testing.T) {
	defer os.Setenv("http_proxy", os.Getenv("http_proxy"))
	defer os.Setenv("https_proxy", os.Getenv("https_proxy"))
	defer os.Setenv("no_proxy", os.Getenv("no_proxy"))
	defer os.Setenv("HTTP_PROXY", os.Getenv("HTTP_PROXY"))
	defer os.Setenv("HTTPS_PROXY", os.Getenv("HTTPS_PROXY"))
	defer os.Setenv("NO_PROXY", os.Getenv("NO_PROXY"))

	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")
	os.Unsetenv("no_proxy")
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")

	server := makeServerConfig("EndpointServer", "http://www.backend.com")
	backEnd := makeBackend(backendA, server)
	testRoute := route{
		Name:     "route1",
		Backends: []*backend{backEnd},
	}

	guardAndHandler := makeGHEntryForSingleBackendRoute(testRoute)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "http://www.input.com", nil)
	assert.Nil(t, err)

	guardAndHandler.HandlerFn.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMakeGHEntryForMultipleBackendsProxy(t *testing.T) {
	defer os.Setenv("http_proxy", os.Getenv("http_proxy"))
	defer os.Setenv("https_proxy", os.Getenv("https_proxy"))
	defer os.Setenv("no_proxy", os.Getenv("no_proxy"))
	defer os.Setenv("HTTP_PROXY", os.Getenv("HTTP_PROXY"))
	defer os.Setenv("HTTPS_PROXY", os.Getenv("HTTPS_PROXY"))
	defer os.Setenv("NO_PROXY", os.Getenv("NO_PROXY"))

	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")
	os.Unsetenv("no_proxy")
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("PROXY REACHED"))
	}))
	defer proxyServer.Close()

	os.Setenv("http_proxy", proxyServer.URL)

	var bHandler plugin.MultiBackendHandlerFunc = func(bhMap plugin.BackendHandlerMap, w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, 2, len(bhMap))

		for _, backendHandler := range bhMap {
			w := httptest.NewRecorder()
			backendHandler.ServeHTTP(w, r)
			assert.Equal(t, http.StatusTeapot, w.Code)
			assert.Equal(t, "PROXY REACHED", w.Body.String())
		}

		w.WriteHeader(http.StatusBadRequest)
	}

	var BMBAFactory = func(bhMap plugin.BackendHandlerMap) *plugin.MultiBackendAdapter {
		return &plugin.MultiBackendAdapter{
			BackendHandlerCtx: bhMap,
			Handler:           bHandler,
		}
	}

	plugin.RegisterMultiBackendAdapterFactory(multiBackendAdapterFactory, BMBAFactory)
	serverA := makeServerConfig("EndpointServer1", "http://www.backendA.com")
	backEndA := makeBackend(backendA, serverA)
	serverB := makeServerConfig("EndpointServer2", "http://www.backendB.com")
	backEndB := makeBackend(backendB, serverB)
	testRoute := route{
		Name:                   "route1",
		Backends:               []*backend{backEndA, backEndB},
		MultiBackendPluginName: multiBackendAdapterFactory,
	}

	guardAndHandler := makeGHEntryForMultipleBackends(testRoute)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "http://www.input.com", nil)
	assert.Nil(t, err)

	guardAndHandler.HandlerFn.ServeHTTP(w, req)
}

func TestMakeGHEntryForMultipleBackendsNoProxy(t *testing.T) {
	defer os.Setenv("http_proxy", os.Getenv("http_proxy"))
	defer os.Setenv("https_proxy", os.Getenv("https_proxy"))
	defer os.Setenv("no_proxy", os.Getenv("no_proxy"))
	defer os.Setenv("HTTP_PROXY", os.Getenv("HTTP_PROXY"))
	defer os.Setenv("HTTPS_PROXY", os.Getenv("HTTPS_PROXY"))
	defer os.Setenv("NO_PROXY", os.Getenv("NO_PROXY"))

	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")
	os.Unsetenv("no_proxy")
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")

	var bHandler plugin.MultiBackendHandlerFunc = func(bhMap plugin.BackendHandlerMap, w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, 2, len(bhMap))

		for _, backendHandler := range bhMap {
			w := httptest.NewRecorder()
			backendHandler.ServeHTTP(w, r)
			assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		}

		w.WriteHeader(http.StatusBadRequest)
	}

	var BMBAFactory = func(bhMap plugin.BackendHandlerMap) *plugin.MultiBackendAdapter {
		return &plugin.MultiBackendAdapter{
			BackendHandlerCtx: bhMap,
			Handler:           bHandler,
		}
	}

	plugin.RegisterMultiBackendAdapterFactory(multiBackendAdapterFactory, BMBAFactory)
	serverA := makeServerConfig("EndpointServer1", "http://www.backendA.com")
	backEndA := makeBackend(backendA, serverA)
	serverB := makeServerConfig("EndpointServer2", "http://www.backendB.com")
	backEndB := makeBackend(backendB, serverB)
	testRoute := route{
		Name:                   "route1",
		Backends:               []*backend{backEndA, backEndB},
		MultiBackendPluginName: multiBackendAdapterFactory,
	}

	guardAndHandler := makeGHEntryForMultipleBackends(testRoute)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "http://www.input.com", nil)
	assert.Nil(t, err)

	guardAndHandler.HandlerFn.ServeHTTP(w, req)
}
