package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequiredConfirmationsOption(t *testing.T) {
	result, err := parseRequiredConfirmationsOption("0.1:1,1.0:2,2.0:6")
	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Equal(t, []RequiredConfirmations{
		{minValSats: 10000000, confirmations: 1},
		{minValSats: 100000000, confirmations: 2},
		{minValSats: 200000000, confirmations: 6},
	}, result)

	require.Equal(t, uint64(1), getRequiredConfirmations(result, 10000000))
	require.Equal(t, uint64(1), getRequiredConfirmations(result, 10000001))
	require.Equal(t, uint64(1), getRequiredConfirmations(result, 99999999))
	require.Equal(t, uint64(2), getRequiredConfirmations(result, 100000000))
	require.Equal(t, uint64(2), getRequiredConfirmations(result, 100000001))
	require.Equal(t, uint64(2), getRequiredConfirmations(result, 199999999))
	require.Equal(t, uint64(6), getRequiredConfirmations(result, 200000000))
	require.Equal(t, uint64(6), getRequiredConfirmations(result, 999999999))
}
