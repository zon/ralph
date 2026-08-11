package trailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatWithKey(t *testing.T) {
	assert.Equal(t, "Ralph item 3 (csv-serializer) completed", Format(3, "csv-serializer"))
}

func TestFormatWithoutKey(t *testing.T) {
	assert.Equal(t, "Ralph item 0 completed", Format(0, ""))
}

func TestFormatRendersIndex(t *testing.T) {
	assert.Equal(t, "Ralph item 2 completed", Format(2, ""))
}
