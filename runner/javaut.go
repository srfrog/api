package runner

import (
	"archive/tar"
	"bytes"
	"encoding/xml"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/coduno/api/model"
	"github.com/coduno/api/util"
	"github.com/fsouza/go-dockerclient"
	"golang.org/x/net/context"
)

func JUnit(ctx context.Context, testFile string, sub model.KeyedSubmission, ball io.Reader) (*model.JunitTestResult, error) {
	image := newImage("javaut")

	if err := prepareImage(image); err != nil {
		return nil, err
	}

	c, err := itoc(image)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		rc, err := util.Load(ctx, util.TestsBucket, testFile)
		if err != nil {
			return
		}
		defer rc.Close()
		buf, err := ioutil.ReadAll(rc)
		if err != nil {
			return
		}
		w := tar.NewWriter(pw)
		defer w.Close()
		w.WriteHeader(&tar.Header{
			Name: path.Base(testFile),
			Mode: 0600,
			Size: int64(len(buf)),
		})
		w.Write(buf)
	}()

	err = dc.UploadToContainer(c.ID, docker.UploadToContainerOptions{
		Path:        "/run/src/test/java/",
		InputStream: pr,
	})
	if err != nil {
		return nil, err
	}

	err = dc.UploadToContainer(c.ID, docker.UploadToContainerOptions{
		Path:        "/run/src/main/java",
		InputStream: ball,
	})
	if err != nil {
		return nil, err
	}

	start := time.Now()
	if err := dc.StartContainer(c.ID, c.HostConfig); err != nil {
		return nil, err
	}

	if err := waitForContainer(c.ID); err != nil {
		return nil, err
	}
	end := time.Now()

	stdout, stderr, err := getLogs(c.ID)
	if err != nil {
		return nil, err
	}

	errc := make(chan error)
	dpr, dpw := io.Pipe()

	go func() {
		errc <- dc.DownloadFromContainer(c.ID, docker.DownloadFromContainerOptions{
			Path:         util.JUnitResultsPath,
			OutputStream: dpw,
		})
	}()

	// TODO(flowlo): encoding/xml can only parse XML 1.0, but JUnit
	// reports are XML 1.1. It appears the reports are valid XML 1.0
	// too, so this replaces the version attribute.
	// As soon as encoding/xml can parse XML 1.1 we can remove this
	// and directly stream without buffering.
	buf, err := ioutil.ReadAll(dpr)
	if err != nil {
		return nil, err
	}
	buf = bytes.Replace(buf, []byte(`version="1.1"`), []byte(`version="1.0"`), 1)

	tr := tar.NewReader(bytes.NewReader(buf))
	d := xml.NewDecoder(tr)

	var testResults *model.JunitTestResult
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(h.Name, ".xml") {
			continue
		}

		var utr model.UnitTestResults
		if err := d.Decode(&utr); err != nil {
			return nil, err
		}
		testResults = &model.JunitTestResult{
			Stdout:  stdout.String(),
			Results: utr,
			Stderr:  stderr.String(),
			Start:   start,
			End:     end,
		}
	}

	return testResults, <-errc
}
