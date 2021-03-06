// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package timeutils

import (
	"testing"
	"time"
)

func TestTimeUtils(t *testing.T) {
	tm := time.Now().UTC()
	tmLocal := time.Now()

	t.Logf("isoTime: %s", IsoTime(tm))
	t.Logf("isoNoSecondTime: %s", IsoNoSecondTime(time.Time{}))
	t.Logf("FullIsoTime: %s", FullIsoTime(tm))
	t.Logf("mysqlTime: %s", MysqlTime(tm))
	t.Logf("CompactTime: %s", CompactTime(tm))
	t.Logf("ShortDate: %s", ShortDate(tm))
	t.Logf("Date: %s", DateStr(tm))
	t.Logf("RFC2882: %s", RFC2882Time(tm))
	t.Logf("ZStack: %s", ZStackTime(tmLocal))
	t.Logf("FullIsoNanoTime: %s", FullIsoNanoTime(tmLocal))

	tm2, err := ParseTimeStr(IsoTime(tm))
	if err != nil {
		t.Errorf("Parse time str error: %s", err)
	}
	tm3, err := ParseTimeStr(MysqlTime(tm))
	if err != nil {
		t.Errorf("Parse time str error: %s", err)
	}
	tm4, err := ParseTimeStr(CompactTime(tm))
	if err != nil {
		t.Errorf("Parse time str error: %s", err)
	}
	tm5, err := ParseTimeStr(ZStackTime(tmLocal))
	if err != nil {
		t.Errorf("Parse time str error: %s", err)
	}
	tm6, err := ParseTimeStr(FullIsoNanoTime(tmLocal))
	if err != nil {
		t.Errorf("Parse time str error: %s", err)
	}

	if tm2 != tm3 || tm2 != tm4 {
		t.Errorf("Parse Iso time error! %s %s", tm, tm2)
	}

	if tmLocal.Sub(tm5) > 1*time.Second {
		t.Errorf("Parse ZStack time error! %s %s %s", tmLocal, tm5, tmLocal.Sub(tm5))
	}
	if tmLocal.Sub(tm6) > 1*time.Second {
		t.Errorf("Parse ZStack time error! %s %s %s", tmLocal, tm5, tmLocal.Sub(tm6))
	}

	cases := []struct {
		in   string
		want string
	}{
		{in: "2019-09-17T03:02:45.709546502+08:00", want: "2019-09-17 03:02:45.709546502 +0800 CST"},
		{in: "2019-09-17T03:15:42.480940759+08:00\n", want: "2019-09-17 03:15:42.480940759 +0800 CST"},
		{in: "2019-09-03T11:25:26.81415Z\n", want: "2019-09-03 11:25:26.81415 +0000 UTC"},
		{in: "2019-09-03T11:25:26.8141523Z\n", want: "2019-09-03 11:25:26.8141523 +0000 UTC"},
		{in: "2019-11-19T18:54:48.084-08:00", want: "2019-11-19 18:54:48.084 -0800 -0800"},
	}
	for _, c := range cases {
		tm, err := ParseTimeStr(c.in)
		if err != nil {
			t.Fatalf("%s fail: %v", c.in, err)
		}
		if tm.String() != c.want {
			t.Fatalf("%s != %s", tm.String(), c.want)
		}
	}
}

func TestToFullIsoNanoTimeFormat(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "2019-09-17T20:50:17.66667134+08:00",
			want: "2019-09-17T20:50:17.666671340+08:00",
		},
		{
			in:   "2019-09-17T20:50:17.66134+08:00",
			want: "2019-09-17T20:50:17.661340000+08:00",
		},
	}
	for _, c := range cases {
		got := toFullIsoNanoTimeFormat(c.in)
		if got != c.want {
			t.Errorf("want %s != got %s", c.want, got)
		}
	}
}
