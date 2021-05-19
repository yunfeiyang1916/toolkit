package gid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUUID(t *testing.T) {
	u := UUID()
	assert.NotNil(t, u)
}
