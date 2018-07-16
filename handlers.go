package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type registryController struct {
	reg  *registry.Registry
	cl   *clair.Clair
	l    sync.Mutex
	tmpl *template.Template
}

type v1Compatibility struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
}

// A Repository holds data after a vulnerability scan of a single repo
type Repository struct {
	Name                string                    `json:"name"`
	Tag                 string                    `json:"tag"`
	Created             time.Time                 `json:"created"`
	URI                 string                    `json:"uri"`
	VulnerabilityReport clair.VulnerabilityReport `json:"vulnerability"`
}

// A AnalysisResult holds all vulnerabilities of a scan
type AnalysisResult struct {
	Repositories   []Repository `json:"repositories"`
	RegistryDomain string       `json:"registryDomain"`
	Name           string       `json:"name"`
	LastUpdated    string       `json:"lastUpdated"`
}

func (rc *registryController) repositories(staticDir string, generateTagsFiles bool) error {
	rc.l.Lock()
	defer rc.l.Unlock()

	logrus.Infof("fetching catalog for %s...", rc.reg.Domain)

	result := AnalysisResult{
		RegistryDomain: rc.reg.Domain,
		LastUpdated:    time.Now().Local().Format(time.RFC1123),
	}

	repoList, err := rc.reg.Catalog("")
	if err != nil {
		return fmt.Errorf("getting catalog for %s failed: %v", rc.reg.Domain, err)
	}

	for _, repo := range repoList {
		repoURI := fmt.Sprintf("%s/%s", rc.reg.Domain, repo)
		r := Repository{
			Name: repo,
			URI:  repoURI,
		}

		result.Repositories = append(result.Repositories, r)

		if generateTagsFiles {
			logrus.Infof("generating static tags page for repo %s", repo)

			// Parse and execute the tags templates.
			b, err := rc.generateTagsTemplate(repo)
			if err != nil {
				logrus.Warnf("generating tags tamplate for repo %q failed: %v", repo, err)
			}
			// Create the directory for the static tags files.
			tagsDir := filepath.Join(staticDir, "repo", repo, "tags")
			if err := os.MkdirAll(tagsDir, 0755); err != nil {
				return err
			}

			// Write the tags file.
			tagsFile := filepath.Join(tagsDir, "index.html")
			if err := ioutil.WriteFile(tagsFile, b, 0755); err != nil {
				logrus.Warnf("writing tags tamplate for repo %s to %sfailed: %v", repo, tagsFile, err)
			}
		}
	}

	// Parse & execute the template.
	logrus.Info("executing the template repositories")

	// Create the static directory.
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return err
	}

	// Creating the index file.
	path := filepath.Join(staticDir, "index.html")
	logrus.Debugf("creating/opening file %s", path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Execute the template on the index.html file.
	if err := rc.tmpl.ExecuteTemplate(f, "repositories", result); err != nil {
		f.Close()
		return fmt.Errorf("execute template repositories failed: %v", err)
	}

	return nil
}

func (rc *registryController) tagsHandler(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"func":   "tags",
		"URL":    r.URL,
		"method": r.Method,
	}).Info("fetching tags")

	// Parse the query variables.
	vars := mux.Vars(r)
	repo, err := url.QueryUnescape(vars["repo"])
	if err != nil || repo == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Empty repo")
		return
	}

	// Generate the tags template.
	b, err := rc.generateTagsTemplate(repo)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func":   "tags",
			"URL":    r.URL,
			"method": r.Method,
		}).Errorf("getting tags for %s failed: %v", repo, err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Getting tags for %s failed", repo)
		return
	}

	// Write the template.
	fmt.Fprint(w, string(b))
}

func (rc *registryController) generateTagsTemplate(repo string) ([]byte, error) {
	// Get the tags from the server.
	tags, err := rc.reg.Tags(repo)
	if err != nil {
		return nil, fmt.Errorf("getting tags for %s failed: %v", repo, err)
	}

	// Error out if there are no tags / images
	// (the above err != nil does not error out when nothing has been found)
	if len(tags) == 0 {
		return nil, fmt.Errorf("no tags found for repo: %s", repo)
	}

	result := AnalysisResult{
		RegistryDomain: rc.reg.Domain,
		LastUpdated:    time.Now().Local().Format(time.RFC1123),
		Name:           repo,
	}

	for _, tag := range tags {
		// get the manifest
		m1, err := rc.reg.ManifestV1(repo, tag)
		if err != nil {
			return nil, fmt.Errorf("getting v1 manifest for %s:%s failed: %v", repo, tag, err)
		}

		var createdDate time.Time
		for _, h := range m1.History {
			var comp v1Compatibility

			if err := json.Unmarshal([]byte(h.V1Compatibility), &comp); err != nil {
				return nil, fmt.Errorf("unmarshal v1 manifest for %s:%s failed: %v", repo, tag, err)
			}

			createdDate = comp.Created
			break
		}

		repoURI := fmt.Sprintf("%s/%s", rc.reg.Domain, repo)
		if tag != "latest" {
			repoURI += ":" + tag
		}
		rp := Repository{
			Name:    repo,
			Tag:     tag,
			URI:     repoURI,
			Created: createdDate,
		}

		result.Repositories = append(result.Repositories, rp)
	}

	// Execute the template.
	var buf bytes.Buffer
	if err := rc.tmpl.ExecuteTemplate(&buf, "tags", result); err != nil {
		return nil, fmt.Errorf("template rendering failed: %v", err)
	}

	return buf.Bytes(), nil
}

func (rc *registryController) vulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"func":   "vulnerabilities",
		"URL":    r.URL,
		"method": r.Method,
	}).Info("fetching vulnerabilities")

	// Parse the query variables.
	vars := mux.Vars(r)
	repo, err := url.QueryUnescape(vars["repo"])
	tag := vars["tag"]

	if err != nil || repo == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Empty repo")
		return
	}

	if tag == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Empty tag")
		return
	}

	result := clair.VulnerabilityReport{}

	if rc.cl != nil {
		result, err = rc.cl.Vulnerabilities(rc.reg, repo, tag)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"func":   "vulnerabilities",
				"URL":    r.URL,
				"method": r.Method,
			}).Errorf("vulnerability scanning for %s:%s failed: %v", repo, tag, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if strings.HasSuffix(r.URL.String(), ".json") {
		js, err := json.Marshal(result)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"func":   "vulnerabilities",
				"URL":    r.URL,
				"method": r.Method,
			}).Errorf("json marshal failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}

	// Execute the template.
	if err := rc.tmpl.ExecuteTemplate(w, "vulns", result); err != nil {
		logrus.WithFields(logrus.Fields{
			"func":   "vulnerabilities",
			"URL":    r.URL,
			"method": r.Method,
		}).Errorf("template rendering failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
