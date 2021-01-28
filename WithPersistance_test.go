package concurrentList

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestWithPersistance(t *testing.T) {
	type test struct {
		Time time.Time
		Data string
	}

	// tempDir, err := ioutil.TempDir(os.TempDir(), "concurrentListWithPersistance")
	// require.NoError(t, err)
	tempDir := filepath.Join(os.TempDir(), "hello")
	_ = os.MkdirAll(tempDir, 0744)
	fmt.Println("using dir ", tempDir)

	list := NewConcurrentList(WithPersistance(tempDir, test{}, func(item interface{}) string {
		return item.(test).Time.Format(time.RFC3339)
	}, func(err error) {
		require.NoError(t, err)
	}))

	fmt.Println("reconstructedList")
	all := list.GetWithFilter(func(item interface{}) bool { return true })
	spew.Dump(all)

	list.Push(test{Time: time.Now(), Data: "firstPush"})

	require.NoError(t, os.RemoveAll(tempDir))
}
