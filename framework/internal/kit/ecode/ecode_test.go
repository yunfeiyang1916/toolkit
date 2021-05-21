package ecode

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yunfeiyang1916/toolkit/framework/ratelimit"
)

func TestConvertHttpStatus(t *testing.T) {
	assert.Equal(t, ConvertHttpStatus(ratelimit.ErrLimited), 501)
	assert.Equal(t, ConvertHttpStatus(nil), 200)
	assert.Equal(t, ConvertHttpStatus(context.DeadlineExceeded), 500)
}
