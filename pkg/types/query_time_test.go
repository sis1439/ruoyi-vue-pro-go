package types

import (
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestQueryTimeRangeFrontEncodings(t *testing.T) {
	for _, s := range []string{"times=A&times=B", "times[]=A&times[]=B", "times[0]=A&times[1]=B", "times=A,B"} {
		v, err := url.ParseQuery(s)
		require.NoError(t, err)
		require.Equal(t, []string{"A", "B"}, QueryTimeRange(v, "times"))
	}
	for _, s := range []string{"times=A", "times=A,B,C", "times[0]=A"} {
		v, err := url.ParseQuery(s)
		require.NoError(t, err)
		require.Nil(t, QueryTimeRange(v, "times"))
	}
}
