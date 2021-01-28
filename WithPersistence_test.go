package concurrentList

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithPersistence(t *testing.T) {
	type test struct {
		Time time.Time
		Data string
	}

	tempDir := filepath.Join(os.TempDir(), "TestWithPersistence")
	_ = os.MkdirAll(tempDir, 0744)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()

	list := NewConcurrentList(WithPersistence(tempDir, test{}, 5*time.Second, func(item interface{}) string {
		return item.(test).Time.Format(time.RFC3339Nano)
	}, func(err error) {
		require.NoError(t, err)
	}), WithSorting(func(i, j interface{}) bool {
		return i.(test).Time.After(j.(test).Time)
	}))

	files, err := ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 0)

	list.Push(test{Time: time.Now(), Data: "firstPush"})
	files, err = ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	list.Push(test{Time: time.Now(), Data: "secondPush"})
	files, err = ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	_, err = list.GetNext(context.Background())
	require.NoError(t, err)
	files, err = ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	list.Push(test{Time: time.Now(), Data: "thirdPush"})
	files, err = ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	_ = list.DeleteWithFilter(func(item interface{}) bool { return true })
	files, err = ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 0)

	list.Push(test{Time: time.Now(), Data: "fourthPush"})
	files, err = ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	list.Push(test{Time: time.Now(), Data: "fifthPush"})
	files, err = ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	list = nil

	// Check if reconstructing the list from file-backup works
	list2 := NewConcurrentList(WithPersistence(tempDir, test{}, 5*time.Second, func(item interface{}) string {
		return item.(test).Time.Format(time.RFC3339Nano)
	}, func(err error) {
		require.NoError(t, err)
	}), WithSorting(func(i, j interface{}) bool {
		return i.(test).Time.After(j.(test).Time)
	}))
	singleItem, err := list2.GetNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, "fourthPush", singleItem.(test).Data)

	singleItem, err = list2.GetNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, "fifthPush", singleItem.(test).Data)
}
