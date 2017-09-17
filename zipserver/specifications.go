package zipserver

import (
	"fmt"
	"net/http"
	"path"
	"strings"
)

// ResourceSpec contains all the info for an HTTP resource relevant for
// setting http headers and keeping track of the extraction work
type ResourceSpec struct {
	size            uint64
	key             string
	contentType     string
	contentEncoding string
}

func (rs *ResourceSpec) String() string {
	formattedEncoding := ""
	if rs.contentEncoding != "" {
		formattedEncoding = fmt.Sprintf(", %s encoding", rs.contentEncoding)
	}

	return fmt.Sprintf("%s (%s%s)", rs.key, rs.contentType, formattedEncoding)
}

// setupRequest sets the proper HTTP headers on a request for storing this resource
func (rs *ResourceSpec) setupRequest(req *http.Request) error {
	// All extracted files must be readable without authentication
	req.Header.Set("x-goog-acl", "public-read")

	req.Header.Set("content-type", rs.contentType)
	if rs.contentEncoding != "" {
		req.Header.Set("content-encoding", rs.contentEncoding)
	}
	return nil
}

// ContentEncodingSpec contains rules for setting the 'Content-Encoding' HTTP header
type ContentEncodingSpec struct {
	extension string
	encoding  string
}

var contentEncodingSpecs = []ContentEncodingSpec{
// // Unity <= 5.5
// {".jsgz", "gzip"},
// {".datagz", "gzip"},
// {".memgz", "gzip"},
// {".unity3dgz", "gzip"},
// // Unity >= 5.6
// {".unityweb", "gzip"},
}

func (rs *ResourceSpec) applyContentEncodingRules() {
	extension := path.Ext(rs.key)

	for _, spec := range contentEncodingSpecs {
		if extension == spec.extension {
			rs.contentEncoding = spec.encoding
			break
		}
	}

	if rs.contentEncoding == "gzip" {
		rs.contentType = "application/octet-stream"
	}
}

// RewriteSpec contains rules for rewriting file extensions
type RewriteSpec struct {
	oldExtension string
	newExtension string
}

var rewriteSpecs = []RewriteSpec{
// // For Unity WebGL up to 5.5, see
// // https://docs.unity3d.com/550/Documentation/Manual/webgl-deploying.html
// {".jsgz", ".js"},
// {".datagz", ".data"},
// {".memgz", ".mem"},
// {".unity3dgz", ".unity3d"},
}

func (rs *ResourceSpec) applyRewriteRules() {
	extension := path.Ext(rs.key)

	for _, spec := range rewriteSpecs {
		if extension == spec.oldExtension {
			rs.key = strings.TrimSuffix(rs.key, spec.oldExtension) + spec.newExtension
			// only apply one rule at most
			return
		}
	}
}
