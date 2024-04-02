package v2

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
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

	list := NewConcurrentList(WithPersistence(tempDir, func(item test) string {
		return item.Time.Format(time.RFC3339Nano)
	}), WithSorting(func(i, j test) bool {
		return i.Time.After(j.Time)
	}))

	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 0)

	list.Push(test{Time: time.Now(), Data: "firstPush"})
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	list.Push(test{Time: time.Now(), Data: "secondPush"})
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	_, err = list.GetNext(context.Background())
	require.NoError(t, err)
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	list.Push(test{Time: time.Now(), Data: "thirdPush"})
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	_ = list.DeleteWithFilter(func(item test) bool { return true })
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 0)

	list.Push(test{Time: time.Now().Add(time.Second), Data: "fourthPush"})
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	list.Push(test{Time: time.Now(), Data: "fifthPush"})
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	list = nil

	// Check if reconstructing the list from file-backup works
	list2 := NewConcurrentList(WithPersistence(tempDir, func(item test) string {
		return item.Time.Format(time.RFC3339Nano)
	}), WithSorting(func(i, j test) bool {
		return i.Time.After(j.Time)
	}))
	singleItem, err := list2.GetNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, "fourthPush", singleItem.Data)

	singleItem, err = list2.GetNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, "fifthPush", singleItem.Data)
}

func TestWithPersistenceOrdering(t *testing.T) {
	type test struct {
		Serial int
		Data   string
	}

	tempDir := filepath.Join(os.TempDir(), "TestWithPersistence")
	_ = os.MkdirAll(tempDir, 0744)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()

	list := NewConcurrentList(WithPersistence(tempDir, func(item test) string {
		return item.Data
	}), WithSorting(func(i, j test) bool {
		return i.Serial < j.Serial
	}))

	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 0)

	list.Push(test{Serial: 2, Data: "aPush"})
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	list.Push(test{Serial: 1, Data: "bPush"})
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	list = nil

	// Check if reconstructing the list from file-backup works
	list2 := NewConcurrentList(WithPersistence(tempDir, func(item test) string {
		return strconv.Itoa(item.Serial)
	}), WithSorting(func(i, j test) bool {
		return i.Serial < j.Serial
	}))

	singleItem, err := list2.GetNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, "bPush", singleItem.Data)

	singleItem, err = list2.GetNext(context.Background())
	require.NoError(t, err)
	require.Equal(t, "aPush", singleItem.Data)
}
