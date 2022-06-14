package amboss

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnection(t *testing.T) {
	magma := NewClient("https://api.amboss.space/graphql", "", 2, 10)
	hello, err := magma.Helloworld()
	if err != nil {
		log.Fatalf("Error. %v", err)
	}
	require.NoError(t, err)
	require.Equal(t, hello, "Hello")
}
