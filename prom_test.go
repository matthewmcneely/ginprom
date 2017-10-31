package ginprom

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/appleboy/gofight"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func unregister(p *Prometheus) {
	prometheus.Unregister(p.reqCnt)
	prometheus.Unregister(p.reqDur)
	prometheus.Unregister(p.reqSz)
	prometheus.Unregister(p.resSz)
}

func TestPrometheus_Use(t *testing.T) {
	p := New()
	r := gin.New()

	p.Use(r)

	assert.Equal(t, 1, len(r.Routes()), "only one route should be added")
	assert.NotNil(t, p.Engine, "the engine should not be empty")
	assert.Equal(t, r, p.Engine, "used router should be the same")
	assert.Equal(t, r.Routes()[0].Path, p.MetricsPath, "the path should match the metrics path")
	unregister(p)
}

func TestPath(t *testing.T) {
	p := New()
	assert.Equal(t, p.MetricsPath, defaultPath, "no usage of path should yield default path")
	unregister(p)

	valid := []string{"/metrics", "/home", "/x/x", ""}
	for _, tt := range valid {
		p = New(Path(tt))
		assert.Equal(t, p.MetricsPath, tt)
		unregister(p)
	}
}

func TestEngine(t *testing.T) {
	r := gin.New()
	p := New(Engine(r))
	assert.Equal(t, 1, len(r.Routes()), "only one route should be added")
	assert.NotNil(t, p.Engine, "engine should not be nil")
	assert.Equal(t, r.Routes()[0].Path, p.MetricsPath, "the path should match the metrics path")
	assert.Equal(t, p.MetricsPath, defaultPath, "path should be default")
	unregister(p)
}

func TestSubsystem(t *testing.T) {
	p := New()
	assert.Equal(t, p.Subsystem, defaultSys, "subsystem should be default")
	unregister(p)

	tests := []string{
		"test",
		"",
		"_",
	}
	for _, test := range tests {
		p = New(Subsystem(test))
		assert.Equal(t, p.Subsystem, test, "should match")
		unregister(p)
	}
}

func TestUse(t *testing.T) {
	r := gin.New()
	p := New()

	g := gofight.New()
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusNotFound, r.Code)
	})

	p.Use(r)
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})
	unregister(p)
}

func TestInstrument(t *testing.T) {
	r := gin.New()
	p := New(Engine(r))
	r.Use(p.Instrument())
	path := "/user/:id"
	lpath := fmt.Sprintf(`path="%s"`, path)

	r.GET(path, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})

	g := gofight.New()
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.NotContains(t, r.Body.String(), `requests_total`)
		assert.NotContains(t, r.Body.String(), lpath, "path must not be present in the response")
	})

	g.GET("/user/10").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusOK, r.Code) })

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.Contains(t, r.Body.String(), `requests_total`)
		assert.Contains(t, r.Body.String(), lpath, "path must be present in the response")
		assert.NotContains(t, r.Body.String(), `path="/user/10"`, "raw path must not be present")
	})
	unregister(p)
}

func TestGetPathFromHandler(t *testing.T) {
	r := gin.New()
	p := New(Engine(r))
	r.Use(p.Instrument())
	path := "/user/:id"
	var hname string
	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
		hname = c.HandlerName()
	}
	// The route has not been added yet so it is unknown to the map
	assert.Empty(t, p.getPathFromHandler(hname))
	r.GET(path, handler)
	// Once there is a request to this route we are forced to update the map
	// thus making it available
	g := gofight.New()
	g.GET("/user/10").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusOK, r.Code) })
	assert.NotEmpty(t, p.getPathFromHandler(hname))
	unregister(p)
}
