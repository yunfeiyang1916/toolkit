package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestJsonEncode(t *testing.T) {
	codec := NewJSONCodec()
	assert := assert.New(t)

	tests := []struct {
		input    Person
		expected string
	}{
		{Person{"tom", 18}, `{"name":"tom","age":18}`},
		{Person{"jack", 22}, `{"name":"jack","age":22}`},
	}
	for _, test := range tests {
		body, err := codec.Encode(test.input)
		assert.Nil(err)
		assert.Equal(string(body), test.expected)
	}
}

func TestJsonDecode(t *testing.T) {
	codec := NewJSONCodec()
	assert := assert.New(t)

	tests := []struct {
		input    string
		expected Person
	}{
		{`{"name":"tom","age":18}`, Person{"tom", 18}},
		{`{"name":"jack","age":22}`, Person{"jack", 22}},
	}
	for _, test := range tests {
		var response Person
		err := codec.Decode([]byte(test.input), &response)
		assert.Nil(err)
		assert.Equal(response, test.expected)
	}
}
