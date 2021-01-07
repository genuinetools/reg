package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/trivy"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type registryController struct {
	reg          *registry.Registry
	cl           *clair.Clair
	trivy        *trivy.Trivy
	interval     time.Duration
	l            sync.Mutex
	tmpl         *template.Template
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

type Repositories []Repository

func (r Repositories) Len() int{
	return len(r)
}

func (r Repositories) Less(i, j int) bool {
	return r[i].Created.Unix() < r[j].Created.Unix()
}

func (r Repositories) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// An AnalysisResult holds all vulnerabilities of a scan
type AnalysisResult struct {
	Repositories   []Repository `json:"repositories"`
	RegistryDomain string       `json:"registryDomain"`
	Name           string       `json:"name"`
	LastUpdated    string       `json:"lastUpdated"`
	HasVulns       bool         `json:"hasVulns"`
	UpdateInterval time.Duration
}

type scanner interface {
	ScanImage(ctx context.Context, r *registry.Registry, repo, tag string) (interface{}, error)
}

func (rc *registryController) currentScanner() scanner {
	if rc.trivy != nil {
		return rc.trivy
	}
	if rc.cl != nil {
		return rc.cl
	}
	return nil
}

func (rc *registryController) repositories(ctx context.Context, staticDir string) error {
	rc.l.Lock()
	defer rc.l.Unlock()

	logrus.Infof("fetching catalog for %s...", rc.reg.Domain)

	result := AnalysisResult{
		RegistryDomain: rc.reg.Domain,
		LastUpdated:    time.Now().Local().Format(time.RFC1123),
		UpdateInterval: rc.interval,
	}

	repoList, err := rc.reg.Catalog(ctx, "")
	if err != nil {
		return fmt.Errorf("getting catalog for %s failed: %v", rc.reg.Domain, err)
	}

	var wg sync.WaitGroup
	for _, repo := range repoList {
		repoURI := fmt.Sprintf("%s/%s", rc.reg.Domain, repo)
		r := Repository{
			Name: repo,
			URI:  repoURI,
		}

		result.Repositories = append(result.Repositories, r)

		// Generate the tags pages in a go routine.
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			logrus.Infof("generating static tags page for repo %s", repo)

			b, tags, err := rc.generateTagsTemplate(ctx, repo, rc.currentScanner() != nil)
			if err != nil {
				logrus.Warnf("generating tags template for repo %q failed: %v", repo, err)
			}
			// Create the directory for the static tags files.
			tagsDir := filepath.Join(staticDir, "repo", repo, "tags")
			if err := os.MkdirAll(tagsDir, 0755); err != nil {
				logrus.Warn(err)
			}

			// Write the tags file.
			tagsFile := filepath.Join(tagsDir, "index.html")
			if err := ioutil.WriteFile(tagsFile, b, 0755); err != nil {
				logrus.Warnf("writing tags template for repo %s to %s failed: %v", repo, tagsFile, err)
			}

			if rc.currentScanner() != nil {
				for _, tag := range tags {
					bvulnhtml, bvulnjson, err := rc.generateVulnerabilityTemplate(ctx, repo, tag, rc.currentScanner())
					if err != nil {
						logrus.Warnf("generating tags template for repo %q failed: %v", repo, err)
					}
					// Create the directory for the static vulnerability files.
					vulnsDir := filepath.Join(staticDir, "repo", repo, "tag", tag.Name, "vulns")
					if err := os.MkdirAll(vulnsDir, 0755); err != nil {
						logrus.Warn(err)
					}

					// Write the vulnerabilies file.
					vulnsFile := filepath.Join(vulnsDir, "index.html")
					if err := ioutil.WriteFile(vulnsFile, bvulnhtml, 0755); err != nil {
						logrus.Warnf("writing vulnerabilities template for repo %s to %s failed: %v", repo, vulnsFile, err)
					}

					// Write the vulnerabilities json
					vulnsJsonFile := filepath.Join(staticDir, "repo", repo, "tag", tag.Name, "vulns.json")
					if err = ioutil.WriteFile(vulnsJsonFile, bvulnjson, 0755); err != nil {
						logrus.Warnf("writing vulnerabilities json template for repo %s to %s failed: %v", repo, vulnsJsonFile, err)
					}
				}
			}
		}(repo)
	}
	wg.Wait()

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
	b, _, err := rc.generateTagsTemplate(context.TODO(), repo, rc.currentScanner() != nil)
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

func (rc *registryController) generateTagsTemplate(ctx context.Context, repo string, hasVulns bool) ([]byte, []registry.Tag, error) {
	// Get the tags from the server.
	tags, err := rc.reg.Tags(ctx, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("getting tags for %s failed: %v", repo, err)
	}

	// Error out if there are no tags / images
	// (the above err != nil does not error out when nothing has been found)
	if len(tags) == 0 {
		return nil, nil, fmt.Errorf("no tags found for repo: %s", repo)
	}

	result := AnalysisResult{
		RegistryDomain: rc.reg.Domain,
		LastUpdated:    time.Now().Local().Format(time.RFC1123),
		UpdateInterval: rc.interval,
		Name:           repo,
		HasVulns:       hasVulns, // if we have a scanner we can return vulns
	}

	for _, tag := range tags {
		createdDate, err := tag.CreatedDate(ctx)
		if err != nil {
			fmt.Errorf("could not get create date. Repo: %s, tag: %s. Error: %v", repo, tag, err)
			continue;
		}
		repoURI := fmt.Sprintf("%s/%s", rc.reg.Domain, repo)
		if tag.Name != "latest" {
			repoURI += ":" + tag.Name
		}
		rp := Repository{
			Name:    repo,
			Tag:     tag.Name,
			URI:     repoURI,
			Created: createdDate,
		}

		result.Repositories = append(result.Repositories, rp)
	}
	sort.Sort(sort.Reverse(Repositories(result.Repositories)))

	// Execute the template.
	var buf bytes.Buffer
	if err := rc.tmpl.ExecuteTemplate(&buf, "tags", result); err != nil {
		return nil, nil, fmt.Errorf("template rendering failed: %v", err)
	}

	return buf.Bytes(), tags, nil
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

	image, err := registry.ParseImage(rc.reg.Domain + "/" + repo + ":" + tag)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func":   "vulnerabilities",
			"URL":    r.URL,
			"method": r.Method,
		}).Errorf("parsing image %s:%s failed: %v", repo, tag, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := rc.currentScanner().ScanImage(context.TODO(), rc.reg, image.Path, image.Reference())
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func":   "vulnerabilities",
			"URL":    r.URL,
			"method": r.Method,
		}).Errorf("vulnerability scanning for %s:%s failed: %v", repo, tag, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
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

func (rc *registryController) generateVulnerabilityTemplate(ctx context.Context, repo string, tag registry.Tag, sc scanner) ([]byte, []byte, error){
	result, err := rc.currentScanner().ScanImage(context.TODO(), rc.reg, repo, tag.Name)
	if err != nil {
		return nil, nil, err
	}
	js, err := json.Marshal(result)
	if err != nil {
		return nil, nil, err
	}
	var html bytes.Buffer
	if err := rc.tmpl.ExecuteTemplate(&html, "vulns", result); err != nil {
		return nil, nil, err
	}
	return html.Bytes(), js, nil
}
