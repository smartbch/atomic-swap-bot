package bot

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type RequiredConfirmations struct {
	minValSats    uint64
	confirmations uint64
}

// eg: "0.1:1,1.0:2,2.0:6"
func parseRequiredConfirmationsOption(option string) ([]RequiredConfirmations, error) {
	var parsed []RequiredConfirmations

	for _, kv := range strings.Split(option, ",") {
		kvSplit := strings.Split(kv, ":")
		if len(kvSplit) != 2 {
			return nil, fmt.Errorf("invalid kv: %s", kv)
		}
		amt, err := strconv.ParseFloat(kvSplit[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amt: %s", kvSplit[0])
		}
		confirmations, err := strconv.ParseUint(kvSplit[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid confirmations: %s", kvSplit[1])
		}
		parsed = append(parsed, RequiredConfirmations{
			minValSats:    uint64(math.Round(amt * 1e8)),
			confirmations: confirmations,
		})
	}
	return parsed, nil
}

func getRequiredConfirmations(s []RequiredConfirmations, amt uint64) uint64 {
	for i := len(s) - 1; i >= 0; i-- {
		if amt >= s[i].minValSats {
			return s[i].confirmations
		}
	}
	return 0
}
