package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	wordwrap "github.com/mitchellh/go-wordwrap"

	"time"

	"net/url"

	"github.com/jessfraz/reg/clair"
	"github.com/jessfraz/reg/registry"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type registryController struct {
	reg *registry.Registry
	cl  *clair.Clair
}

// A Template hold template data
type Template struct {
	templates *template.Template
}

type Repository struct {
	Name                string                    `json:"name"`
	Tag                 string                    `json:"tag"`
	Created             time.Time                 `json:"created"`
	URI                 string                    `json:"uri"`
	VulnerabilityReport clair.VulnerabilityReport `json:"vulnerability"`
}

type AnalysisResult struct {
	Repositories   []Repository `json:"repositories"`
	RegistryDomain string       `json:"registrydomain"`
	Name           string       `json:"name"`
}

// Render a template
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func listenAndServe(port, keyfile, certfile string, r *registry.Registry, c *clair.Clair) error {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Static("static"))

	funcMap := template.FuncMap{
		"trim": func(s string) string {
			return wordwrap.WrapString(s, 80)
		},
		"color": func(s string) string {
			switch s = strings.ToLower(s); s {
			case "high":
				return "danger"
			case "critical":
				return "danger"
			case "defcon1":
				return "danger"
			case "medium":
				return "warning"
			case "low":
				return "info"
			case "negligible":
				return "info"
			case "unknown":
				return "default"
			default:
				return "default"
			}
		},
	}
	// precompile templates
	t := &Template{
		templates: template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/echo/*.html")),
	}
	e.Renderer = t

	rc := registryController{
		reg: r,
		cl:  c,
	}

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/repo")
	})

	e.GET("/repo", rc.repositories)
	e.GET("/repo/:repo", rc.tags)
	e.GET("/repo/:repo/:tag", rc.tag)
	e.GET("/repo/:repo/:tag/vulns", rc.vulnerabilities)

	srv := &http.Server{
		Addr: ":" + port,
	}

	if keyfile != "" && certfile != "" {
		cer, err := tls.LoadX509KeyPair(certfile, keyfile)
		if err != nil {
			return err
		}
		srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cer}}
	}

	return e.StartServer(srv)
}

func (rc *registryController) repositories(c echo.Context) error {
	log.WithFields(log.Fields{
		"method":  "repositories",
		"context": c,
	}).Debug("fetching repositories")

	result := AnalysisResult{}
	result.RegistryDomain = rc.reg.Domain

	repoList, err := rc.reg.Catalog("")
	if err != nil {
		return fmt.Errorf("getting catalog failed: %v", err)
	}
	for _, repo := range repoList {
		log.WithFields(log.Fields{
			"repo": repo,
		}).Debug("fetched repo")
		repoURI := fmt.Sprintf("%s/%s", rc.reg.Domain, repo)
		r := Repository{
			Name: repo,
			URI:  repoURI,
		}

		result.Repositories = append(result.Repositories, r)
	}
	err = c.Render(http.StatusOK, "repositories", result)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error during template rendering")
	}
	return err
}

// e.GET("/repo/:repo/:tag", rc.tag)
func (rc *registryController) tag(c echo.Context) error {
	repo, err := url.QueryUnescape(c.Param("repo"))
	if err != nil {
		return c.String(http.StatusNotFound, "Given repo can not be unescaped.")
	}
	if repo == "" {
		return c.String(http.StatusNotFound, "No repo given")
	}
	tag := c.Param("tag")
	if tag == "" {
		return c.String(http.StatusNotFound, "No tag given")
	}

	return c.String(http.StatusOK, fmt.Sprintf("Repo: %s Tag: %s ", repo, tag))
}

func (rc *registryController) tags(c echo.Context) error {
	repo, err := url.QueryUnescape(c.Param("repo"))
	if err != nil {
		return c.String(http.StatusNotFound, "Given repo can not be unescaped.")
	}
	if repo == "" {
		return c.String(http.StatusNotFound, "No repo given")
	}
	log.WithFields(log.Fields{
		"method":  "tags",
		"context": c,
		"repo":    repo,
	}).Info("fetching tags")

	tags, err := rc.reg.Tags(repo)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"repo":  repo,
		}).Error("getting tags failed.", repo, err)
		return c.String(http.StatusNotFound, "No Tags found")
	}

	result := AnalysisResult{}
	result.RegistryDomain = rc.reg.Domain
	result.Name = repo
	for _, tag := range tags {
		// get the manifest

		m1, err := r.ManifestV1(repo, tag)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"repo":  repo,
				"tag":   tag,
			}).Warn("getting v1 manifest failed")
		}

		var createdDate time.Time
		for _, h := range m1.History {
			var comp v1Compatibility
			if err := json.Unmarshal([]byte(h.V1Compatibility), &comp); err != nil {
				msg := "unmarshal v1compatibility failed"
				log.WithFields(log.Fields{
					"error": err,
					"repo":  repo,
					"tag":   tag,
				}).Warn(msg)
				return c.String(http.StatusInternalServerError, msg)
			}
			createdDate = comp.Created
			break
		}

		repoURI := fmt.Sprintf("%s/%s", r.Domain, repo)
		if tag != "latest" {
			repoURI += ":" + tag
		}
		r := Repository{
			Name:    repo,
			Tag:     tag,
			URI:     repoURI,
			Created: createdDate,
		}

		if rc.cl != nil {
			vuln, err := rc.cl.Vulnerabilities(rc.reg, repo, tag, m1)
			if err != nil {
				msg := "error during vulnerability scanning."
				log.WithFields(log.Fields{
					"error": err,
					"repo":  repo,
					"tag":   tag,
				}).Error(msg)
				return c.String(http.StatusInternalServerError, msg)
			}
			r.VulnerabilityReport = vuln
		}

		result.Repositories = append(result.Repositories, r)
	}
	err = c.Render(http.StatusOK, "tags", result)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error during template rendering")
	}
	return err
}

func (rc *registryController) vulnerabilities(c echo.Context) error {
	repo, err := url.QueryUnescape(c.Param("repo"))
	if err != nil {
		return c.String(http.StatusNotFound, "Given repo can not be unescaped.")
	}
	if repo == "" {
		return c.String(http.StatusNotFound, "No repo given")
	}
	tag := c.Param("tag")
	if tag == "" {
		return c.String(http.StatusNotFound, "No tag given")
	}

	m1, err := r.ManifestV1(repo, tag)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"repo":  repo,
			"tag":   tag,
		}).Warn("getting v1 manifest failed")
	}

	for _, h := range m1.History {
		var comp v1Compatibility
		if err := json.Unmarshal([]byte(h.V1Compatibility), &comp); err != nil {
			msg := "unmarshal v1compatibility failed"
			log.WithFields(log.Fields{
				"error": err,
				"repo":  repo,
				"tag":   tag,
			}).Warn(msg)
			return c.String(http.StatusInternalServerError, msg)
		}
		break
	}

	result := clair.VulnerabilityReport{}

	if rc.cl != nil {
		result, err = rc.cl.Vulnerabilities(rc.reg, repo, tag, m1)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"repo":  repo,
				"tag":   tag,
			}).Error("error during vulnerability scanning.")
		}
	}

	err = c.Render(http.StatusOK, "vulns", result)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error during template rendering")
	}
	return err
}
