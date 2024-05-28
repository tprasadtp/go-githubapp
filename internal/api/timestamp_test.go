// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package api

import (
	"encoding/json"
	"testing"
	"time"
)

const (
	emptyTimeStr         = `"0001-01-01T00:00:00Z"`
	refTimeStr           = `"2006-01-02T15:04:05Z"`
	refTimeStrFractional = `"2006-01-02T15:04:05.000Z"`
)

const (
	refUnixTimeStr             = `1136214245`
	refUnixTimeStrMilliSeconds = `1136214245000`
)

//nolint:gochecknoglobals
var (
	refTimeGo       = time.Date(2006, time.January, 02, 15, 04, 05, 0, time.UTC)
	refTimeUnixZero = time.Unix(0, 0).In(time.UTC)
)

func TestTimestamp_Marshal(t *testing.T) {
	tt := []struct {
		name   string
		data   Timestamp
		expect string
		ok     bool
		equal  bool
	}{
		{"Reference", Timestamp{refTimeGo}, refTimeStr, false, true},
		{"Empty", Timestamp{}, emptyTimeStr, false, true},
		{"Mismatch", Timestamp{}, refTimeStr, false, false},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out, err := json.Marshal(tc.data)
			if ok := err != nil; ok != tc.ok {
				t.Errorf("ok=%v, expected.ok=%v, err=%v", ok, tc.ok, err)
			}
			v := string(out)
			equal := v == tc.expect
			if (v == tc.expect) != tc.equal {
				t.Errorf("got=%s, tc.expect=%s, equal=%v, tc.equal=%v", v, tc.expect, equal, tc.equal)
			}
		})
	}
}

func TestTimestamp_Unmarshal(t *testing.T) {
	type testCase struct {
		name   string
		data   string
		expect Timestamp
		ok     bool
		equal  bool
	}

	tt := []testCase{
		{"ReferenceGo", refTimeStr, Timestamp{refTimeGo}, false, true},
		{"ReferenceUnix", refUnixTimeStr, Timestamp{refTimeGo}, false, true},
		{"ReferenceUnixMillisecond", refUnixTimeStrMilliSeconds, Timestamp{refTimeGo}, false, true},
		{"ReferenceGoFractional", refTimeStrFractional, Timestamp{refTimeGo}, false, true},
		{"Empty", emptyTimeStr, Timestamp{}, false, true},
		{"UnixZero", `0`, Timestamp{refTimeUnixZero}, false, true},
		{"Mismatch", refTimeStr, Timestamp{}, false, false},
		{"MismatchUnix", `0`, Timestamp{}, false, false},
		{"Invalid", `"asdf"`, Timestamp{refTimeGo}, true, false},
		{"OffByMillisecond", `1136214245001`, Timestamp{refTimeGo}, false, false},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var got Timestamp
			err := json.Unmarshal([]byte(tc.data), &got)
			if gotErr := err != nil; gotErr != tc.ok {
				t.Errorf("%s: gotErr=%v, wantErr=%v, err=%v", tc.name, gotErr, tc.ok, err)
			}
			equal := got.Equal(tc.expect)
			if equal != tc.equal {
				t.Errorf("%s: got=%#v, want=%#v, equal=%v, want=%v", tc.name, got, tc.expect, equal, tc.equal)
			}
		})
	}
}

func TestTimestamp_MarshalReflexivity(t *testing.T) {
	type testCase struct {
		name string
		data Timestamp
	}

	tt := []testCase{
		{"Reference", Timestamp{refTimeGo}},
		{"Empty", Timestamp{}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.data)
			if err != nil {
				t.Errorf("%s: Marshal err=%v", tc.name, err)
			}
			var got Timestamp
			err = json.Unmarshal(data, &got)
			if err != nil {
				t.Errorf("%s: Unmarshal err=%v", tc.name, err)
			}
			if !got.Equal(tc.data) {
				t.Errorf("%s: %+v != %+v", tc.name, got, data)
			}
		})
	}
}
