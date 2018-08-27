/*
Sniperkit-Bot
- Status: analyzed
*/

package data

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type DataIndex struct {
	Name string
	Http *HttpClient

	// For now, use S3Store directly. clean up interface later.
	// BlobStore blobStore
	BlobStore *S3Store
}

var mainDataIndex *DataIndex

const mainIndexName = "datadex"

// why not use `func init()`? some commands don't need an index
// is annoying to error out on an S3 key when S3 isn't needed.
func NewMainDataIndex() (*DataIndex, error) {
	if mainDataIndex != nil {
		return mainDataIndex, nil
	}

	i := &DataIndex{Name: mainIndexName}
	err := error(nil)

	i.Http, err = NewHttpClient(i.Name)
	if err != nil {
		return nil, err
	}

	i.BlobStore, err = NewS3Store("datadex.archives", i)
	if err != nil {
		return nil, err
	}

	mainDataIndex = i
	return mainDataIndex, nil
}

const HttpHeaderUser = "X-Data-User"
const HttpHeaderToken = "X-Data-Token"
const HttpHeaderContentType = "Content-Type"
const HttpHeaderContentTypeYaml = "application/yaml"
const ApiUrlSuffix = "/api/v1"

// Controls authenticated http accesses.
type HttpClient struct {
	BaseUrl   string
	Url       string
	User      string
	AuthToken string
}

func NewHttpClient(index string) (*HttpClient, error) {
	i, err := configGetIndex(index)
	if err != nil {
		return nil, err
	}

	h := &HttpClient{
		BaseUrl:   strings.ToLower(i["url"]),
		User:      i["user"],
		AuthToken: i["token"],
	}

	// ensure url has protocol prefix
	if !strings.HasPrefix(h.BaseUrl, "http://") &&
		!strings.HasPrefix(h.BaseUrl, "https://") {
		h.BaseUrl = "http://" + h.BaseUrl
	}
	h.Url = h.BaseUrl

	// ensure url has api suffix
	if !strings.HasSuffix(strings.ToLower(h.Url), ApiUrlSuffix) {
		h.Url = h.Url + ApiUrlSuffix
	}

	return h, nil
}

func (h HttpClient) SubUrl(path string) string {
	return h.Url + "/" + path
}

func (h *HttpClient) Get(path string) (*http.Response, error) {
	dOut("http index get %s\n", h.SubUrl(path))

	req, err := http.NewRequest("GET", h.SubUrl(path), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add(HttpHeaderToken, h.AuthToken)
	req.Header.Add(HttpHeaderUser, h.User)
	return h.DoRequest(req)
}

func (h *HttpClient) Post(path string, body interface{}) (*http.Response, error) {
	dOut("http index post %s\n", h.SubUrl(path))

	rdr := io.Reader(nil)
	var err error
	if body != nil {
		rdr, err = Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", h.SubUrl(path), rdr)
	if err != nil {
		return nil, err
	}

	req.Header.Add(HttpHeaderContentType, HttpHeaderContentTypeYaml)
	req.Header.Add(HttpHeaderToken, h.AuthToken)
	req.Header.Add(HttpHeaderUser, h.User)
	return h.DoRequest(req)
}

func (h *HttpClient) DoRequest(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	c := resp.StatusCode
	if 200 <= c && c < 400 {
		return resp, nil
	}

	e, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	s := strings.TrimSpace(string(e[:]))
	return nil, fmt.Errorf("HTTP error status code: %d (%s)", c, s)
}
