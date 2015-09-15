package runner

import (
	"bytes"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"google.golang.org/cloud/storage"

	"github.com/coduno/api/model"
	"github.com/coduno/api/util"
	"github.com/fsouza/go-dockerclient"
	"golang.org/x/net/context"
)

func diffRunner(ctx context.Context, test *model.Test, sub model.KeyedSubmission) error {
	c, err := dc.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			// TODO(flowlo): Check if the language is known.
			Image: newImage(sub.Language),
		},
		HostConfig: &docker.HostConfig{
			Privileged: false,
			Memory:     0, // TODO(flowlo): Limit memory
		},
	})
	if err != nil {
		return err
	}

	err = dc.StartContainer(c.ID, c.HostConfig)
	if err != nil {
		return err
	}

	stdout, stderr, stdin := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)

	err = dc.AttachToContainer(docker.AttachToContainerOptions{
		Container:    c.ID,
		OutputStream: stdout,
		Stdout:       true,
		Stream:       true,
	})
	if err != nil {
		return err
	}

	err = dc.AttachToContainer(docker.AttachToContainerOptions{
		Container:    c.ID,
		OutputStream: stderr,
		Stderr:       true,
		Stream:       true,
	})
	if err != nil {
		return err
	}

	err = dc.AttachToContainer(docker.AttachToContainerOptions{
		Container:    c.ID,
		OutputStream: stdin,
		Stdin:        true,
		Stream:       true,
	})
	if err != nil {
		return err
	}

	// TODO(flowlo): Save result.
	return nil
}

func IODiffRun(ctx context.Context, in, out string, sub model.KeyedSubmission) (testResult model.DiffTestResult, err error) {
	image := newImage(sub.Language)

	if err = prepareImage(image); err != nil {
		return
	}

	var v *docker.Volume
	if v, err = createDockerVolume(sub.Code.Bucket + "/" + path.Dir(sub.Code.Name)); err != nil {
		return
	}

	var c *docker.Container
	if c, err = createDockerContainer(image, []string{v.Name + ":/run"}); err != nil {
		return
	}

	var stdin io.ReadCloser
	stdin, err = storage.NewReader(util.CloudContext(ctx), util.TestsBucket, in)
	if err != nil {
		return
	}

	start := time.Now()
	if err = dc.StartContainer(c.ID, c.HostConfig); err != nil {
		return
	}

	err = dc.AttachToContainer(docker.AttachToContainerOptions{
		Container:   c.ID,
		InputStream: stdin,
		Stdin:       true,
		Stream:      true,
	})
	if err != nil {
		return
	}

	if err = waitForContainer(c.ID); err != nil {
		return
	}
	end := time.Now()

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	if stdout, stderr, err = getLogs(c.ID); err != nil {
		return
	}

	testResult = model.DiffTestResult{
		SimpleTestResult: model.SimpleTestResult{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
			Start:  start,
			End:    end,
		},
	}

	var want io.ReadCloser
	want, err = storage.NewReader(util.CloudContext(ctx), util.TestsBucket, out)
	if err != nil {
		return
	}

	have := strings.NewReader(testResult.Stdout)
	diffLines, ok, err := compare(want, have)
	if err != nil {
		return
	}
	if !ok {
		return
	}
	testResult.DiffLines = diffLines

	return
}

func OutMatchDiffRun(ctx context.Context, params map[string]string, sub model.KeyedSubmission) (testResult model.DiffTestResult, err error) {
	var str model.SimpleTestResult
	str, err = Simple(ctx, sub)
	if err != nil {
		return
	}
	testResult = model.DiffTestResult{
		SimpleTestResult: str,
	}

	var want io.ReadCloser
	want, err = storage.NewReader(util.CloudContext(ctx), params["bucket"], params["tests"])
	if err != nil {
		return
	}

	have := strings.NewReader(testResult.Stdout)
	diffLines, ok, err := compare(want, have)
	if err != nil {
		return
	}
	if !ok {
		return
	}
	testResult.DiffLines = diffLines
	return
}

func compare(want, have io.Reader) ([]int, bool, error) {
	w, err := ioutil.ReadAll(want)
	if err != nil {
		return nil, false, err
	}
	h, err := ioutil.ReadAll(have)
	if err != nil {
		return nil, false, err
	}
	wb := bytes.Split(w, []byte("\n"))
	hb := bytes.Split(h, []byte("\n"))

	if len(wb) != len(hb) {
		return nil, false, nil
	}

	var diff []int
	for i := 0; i < len(wb); i++ {
		if bytes.Compare(wb[i], hb[i]) != 0 {
			diff = append(diff, i)
		}
	}

	return diff, true, nil
}
