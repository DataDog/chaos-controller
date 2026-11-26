// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cmd

import (
	"strconv"
	"testing"
)

func TestNetworkPercentageInputValidation(t *testing.T) {
	for _, input := range []string{"garbage", "50%", "0.5", " 50", "50 ", "101", "-1", "999999999999999999999999999999999999"} {
		t.Run("reject/"+input, func(t *testing.T) {
			if err := percentageValidator(input); err == nil {
				value, conversionErr := strconv.Atoi(input)
				t.Fatalf("accepted invalid network percentage %q: downstream Atoi returns %d, %v", input, value, conversionErr)
			}
		})
	}

	for _, input := range []string{"0", "1", "50", "100"} {
		t.Run("accept/"+input, func(t *testing.T) {
			if err := percentageValidator(input); err != nil {
				t.Fatalf("rejected valid percentage %q: %v", input, err)
			}
			value, err := strconv.Atoi(input)
			if err != nil || value < 0 || value > 100 {
				t.Fatalf("validated input is not usable by the network builder: %d, %v", value, err)
			}
		})
	}

	// Requiredness is handled by survey.Required at the prompt call sites.
	if err := percentageValidator(""); err != nil {
		t.Fatalf("changed optional empty-input behavior: %v", err)
	}
}
